package service

import (
	ai_client "AI-Dialogue-Map/internal/ai" // Importing ai package for GeminiClient
	"AI-Dialogue-Map/internal/config"
	"AI-Dialogue-Map/internal/ui"
	"AI-Dialogue-Map/internal/utils"
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

const (
	projectsBaseDir = "projects"
	yamlFileName    = "tree.yaml"
	mdNodesDirName  = "nodes"

	nodeSpacing             float32 = 40
	nodeWidthCollapsed      float32 = 220
	nodeHeightCollapsed     float32 = 110
	nodeWidthExpanded       float32 = 380
	maxNodeHeightExpanded   float32 = 600
	maxAnswerLinesCollapsed         = 2
	nodeTitleMaxLength              = 25
)

// TreeData はプロジェクト全体のデータを保持します。
type TreeData struct {
	Nodes       []*ui.NodeData `yaml:"nodes"` // NodeData is defined in node_widget.go
	ProjectName string         `yaml:"project_name"`
}

type App struct {
	fyneApp fyne.App
	window  fyne.Window

	geminiClient *ai_client.GeminiClient
	dialogCanvas *ui.DialogCanvas
	chatInput    *widget.Entry
	sendButton   *widget.Button
	statusLabel  *widget.Label

	nodesMutex         sync.RWMutex // protects nodes
	nodes              []*ui.NodeData
	uiUpdateChan       chan *ui.NodeData
	currentProjectID   string
	currentProjectName string
}

func NewMainApp() *App {
	fyneAppInstance := app.New()
	fyneAppInstance.Settings().SetTheme(ui.NewMyTheme())

	err := config.LoadConfig()
	if err != nil {
		log.Printf("警告: 設定ファイルの読み込みに失敗しました: %v", err)
	}
	window := fyneAppInstance.NewWindow("AI対話ツリー化アプリケーション")
	window.Resize(fyne.NewSize(1200, 800))

	var gemini *ai_client.GeminiClient
	if config.Cfg.GeminiAPIKey != "" {
		gemini, err = ai_client.NewGeminiClient(config.Cfg.GeminiAPIKey)
		if err != nil {
			log.Printf("Geminiクライアントの初期化に失敗しました: %v", err)
		} else {
			log.Println("Geminiクライアントが正常に初期化されました。")
		}
	}

	ma := &App{
		fyneApp:      fyneAppInstance,
		window:       window,
		geminiClient: gemini,
		nodes:        make([]*ui.NodeData, 0),
		uiUpdateChan: make(chan *ui.NodeData, 10),
	}
	ma.updateWindowTitle()

	ma.dialogCanvas = ui.NewDialogCanvas(fyneAppInstance, ma.requestNodeDeletion)
	ma.chatInput = widget.NewMultiLineEntry()
	ma.chatInput.SetPlaceHolder("AIへの質問を入力してください...")
	ma.chatInput.Wrapping = fyne.TextWrapWord
	ma.chatInput.SetMinRowsVisible(3)

	ma.window.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyReturn,
		Modifier: fyne.KeyModifierControl,
	}, func(shortcut fyne.Shortcut) {
		if ma.window.Canvas().Focused() == ma.chatInput {
			log.Println("Ctrl+Return shortcut activated for chatInput")
			ma.handleSend()
		} else {
			log.Println("Ctrl+Return shortcut ignored, chatInput not focused")
		}
	})
	ma.window.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyEnter,
		Modifier: fyne.KeyModifierControl,
	}, func(shortcut fyne.Shortcut) {
		if ma.window.Canvas().Focused() == ma.chatInput {
			log.Println("Ctrl+NumPadEnter shortcut activated for chatInput")
			ma.handleSend()
		} else {
			log.Println("Ctrl+NumPadEnter shortcut ignored, chatInput not focused")
		}
	})

	ma.sendButton = widget.NewButton("送信", ma.handleSend)
	ma.statusLabel = widget.NewLabel("準備完了 (プロジェクトなし)")
	ma.statusLabel.Alignment = fyne.TextAlignCenter

	inputArea := container.NewBorder(nil, nil, nil, ma.sendButton, ma.chatInput)
	bottomBar := container.NewVBox(inputArea, ma.statusLabel)

	split := container.NewVSplit(ma.dialogCanvas, bottomBar)
	split.Offset = 0.85

	window.SetContent(split)
	ma.createMenu()

	window.SetCloseIntercept(func() {
		ma.fyneApp.Quit()
	})

	return ma
}

func (a *App) updateWindowTitle() {
	title := "AI Dialogue Map"
	if a.currentProjectName != "" {
		title = fmt.Sprintf("%s - %s", title, a.currentProjectName)
	} else if a.currentProjectID != "" {
		title = fmt.Sprintf("%s - %s", title, a.currentProjectID)
	}
	a.window.SetTitle(title)
}

func (a *App) createMenu() {
	newProjectItem := fyne.NewMenuItem("新規プロジェクト", a.newProject)
	openProjectItem := fyne.NewMenuItem("プロジェクトを開く...", a.openProjectDialog)
	saveItem := fyne.NewMenuItem("プロジェクトを保存", a.saveCurrentProject)
	exitItem := fyne.NewMenuItem("終了", func() { a.fyneApp.Quit() })
	fileMenu := fyne.NewMenu("ファイル", newProjectItem, openProjectItem, saveItem, fyne.NewMenuItemSeparator(), exitItem)

	branchSourceItem := fyne.NewMenuItem("選択中分岐元表示", func() {
		log.Printf("現在選択中の分岐元ノードID: %s", a.dialogCanvas.GetBranchSource())
		branchSource := a.dialogCanvas.GetBranchSource()
		a.statusLabel.SetText(fmt.Sprintf("選択中: %s", branchSource))
	})
	debugMenu := fyne.NewMenu("デバッグ", branchSourceItem)

	mainMenu := fyne.NewMainMenu(fileMenu, debugMenu)
	a.window.SetMainMenu(mainMenu)
}

func (a *App) getConversationHistory(targetNodeID string) string {
	a.nodesMutex.RLock()
	defer a.nodesMutex.RUnlock()

	var historyParts []string
	currentNodeID := targetNodeID

	nodeDataMap := make(map[string]*ui.NodeData)
	for _, n := range a.nodes {
		nodeDataMap[n.ID] = n
	}

	for currentNodeID != "" {
		currentNodeData, found := nodeDataMap[currentNodeID]
		if !found {
			log.Printf("getConversationHistory: NodeData not found for ID %s", currentNodeID)
			break
		}
		historyParts = append(historyParts, fmt.Sprintf("AI: %s\n", currentNodeData.Answer))
		historyParts = append(historyParts, fmt.Sprintf("User: %s\n", currentNodeData.Question))

		currentNodeID = currentNodeData.ParentID
	}

	var builder strings.Builder
	for i := len(historyParts) - 1; i >= 0; i-- {
		builder.WriteString(historyParts[i])
		if i > 0 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func (a *App) handleSend() {
	currentQuestion := a.chatInput.Text
	if currentQuestion == "" {
		dialog.ShowInformation("情報", "質問を入力してください。", a.window)
		return
	}

	isNewProject := false
	if a.currentProjectID == "" {
		isNewProject = true
	}

	log.Printf("ユーザーからの質問: %s (プロジェクト: %s)", currentQuestion, a.currentProjectID)
	a.chatInput.SetText("")
	a.sendButton.Disable()
	a.statusLabel.SetText("AI応答生成中...")

	branchSource := a.dialogCanvas.GetBranchSource()
	var parentID string
	if branchSource != "" {
		parentID = branchSource
	}

	conversationHistory := ""
	if parentID != "" {
		conversationHistory = a.getConversationHistory(parentID)
		if conversationHistory != "" {
			conversationHistory += "\n\n"
		}
	}
	instructedQuestion := "応答の最初の行に「Title: 」に続けてタイトルを記述し、改行を2つ入れてから本文を記述してください。\n\n質問： " + currentQuestion
	fullPrompt := conversationHistory + "User: " + instructedQuestion

	go func(promptToSend string, originalQuestion string, pID string, isFirstNodeInProject bool) {
		defer func() {
			fyne.Do(func() {
				if a.sendButton != nil {
					a.sendButton.Enable()
				}
				if a.statusLabel != nil {
					a.statusLabel.SetText("準備完了")
				}
			})
		}()

		var answerText string
		var err error

		if a.geminiClient != nil {
			answerText, err = a.geminiClient.Generate(promptToSend)
			if err != nil {
				log.Printf("Gemini API Error: %v", err)
				answerText = fmt.Sprintf("API Error: %v", err)
			}
		} else {
			answerText = fmt.Sprintf("「%s」に対するAIの応答です。(APIキー未設定)", originalQuestion)
			log.Println("Gemini client not initialized.")
		}

		nodeTitle := ""
		nodeAnswerContent := answerText

		if strings.HasPrefix(answerText, "Title: ") {
			parts := strings.SplitN(answerText, "\n", 3)
			if len(parts) >= 1 {
				nodeTitle = strings.TrimSpace(strings.TrimPrefix(parts[0], "Title: "))
				if len(parts) == 2 {
					if strings.TrimSpace(parts[1]) == "" {
						nodeAnswerContent = ""
					} else {
						nodeAnswerContent = parts[1]
					}
				} else if len(parts) >= 3 {
					nodeAnswerContent = parts[2]
				} else {
					nodeAnswerContent = ""
				}
			}
		}

		if nodeTitle == "" {
			log.Println("Title not extracted via 'Title: ' prefix. Using fallback.")
			if firstNewLine := strings.Index(answerText, "\n"); firstNewLine != -1 {
				nodeTitle = answerText[:firstNewLine]
			} else {
				nodeTitle = answerText
			}
		}
		nodeTitle = utils.TruncateText(nodeTitle, nodeTitleMaxLength*2)
		if nodeTitle == "" {
			nodeTitle = "無題のノード"
		}

		if isFirstNodeInProject {
			fyne.Do(func() {
				a.currentProjectID = uuid.NewString()
				a.currentProjectName = nodeTitle
				if a.currentProjectName == "" {
					a.currentProjectName = "New Project - " + time.Now().Format("150405")
				}
				a.updateWindowTitle()
				log.Printf("新規プロジェクトが作成されました: ID=%s, Name=%s", a.currentProjectID, a.currentProjectName)
			})
		}

		newNodeData := &ui.NodeData{
			ID:       uuid.NewString(),
			Title:    nodeTitle,
			Question: originalQuestion,
			Answer:   nodeAnswerContent,
			Expanded: false,
			ParentID: pID,
		}
		a.uiUpdateChan <- newNodeData

	}(fullPrompt, currentQuestion, parentID, isNewProject)
}

func (a *App) handleUIUpdates() {
	for nodeData := range a.uiUpdateChan {
		dataCopy := nodeData
		fyne.Do(func() {
			a.addNode(dataCopy)
			if a.currentProjectID != "" {
				a.saveCurrentProject()
			}
		})
	}
}

func (a *App) addNode(data *ui.NodeData) {
	a.nodesMutex.Lock()
	a.nodes = append(a.nodes, data)
	a.nodesMutex.Unlock()
	a.dialogCanvas.AddNode(data)
	a.dialogCanvas.SetBranchSource(data.ID)
	a.dialogCanvas.Refresh()
}

func (a *App) requestNodeDeletion(nodeID string) {
	log.Printf("App.requestNodeDeletion: %s", nodeID)
	fyne.Do(func() {
		deletedIDs := a.dialogCanvas.RemoveNodeAndDescendants(nodeID)
		a.updateAppDataAfterDeletion(deletedIDs)
		if a.dialogCanvas != nil {
			a.dialogCanvas.Refresh()
		}
		if a.currentProjectID != "" && len(deletedIDs) > 0 {
			a.saveCurrentProject()
		}
	})
}

func (a *App) updateAppDataAfterDeletion(deletedIDs []string) {
	log.Printf("App.updateAppDataAfterDeletion for IDs: %v", deletedIDs)
	a.nodesMutex.Lock()
	defer a.nodesMutex.Unlock()

	if len(deletedIDs) == 0 {
		return
	}

	deletedSet := make(map[string]bool)
	for _, id := range deletedIDs {
		deletedSet[id] = true
	}

	newNodesData := []*ui.NodeData{}
	for _, n := range a.nodes {
		if !deletedSet[n.ID] {
			if n.ParentID != "" && deletedSet[n.ParentID] {
				log.Printf("Clearing ParentID for child node %s (parent %s was deleted)", n.ID, n.ParentID)
				n.ParentID = ""
			}
			newNodesData = append(newNodesData, n)
		}
	}
	a.nodes = newNodesData
	log.Printf("App.nodes (NodeData) updated, count: %d", len(a.nodes))
}

func (a *App) newProject() {
	log.Println("Creating new project.")
	fyne.Do(func() {
		a.clearCurrentProjectState()
		if a.statusLabel != nil {
			a.statusLabel.SetText("新規プロジェクト (名称未設定)")
		}
		a.updateWindowTitle()
		if a.dialogCanvas != nil {
			a.dialogCanvas.Refresh()
		}
	})
}

func (a *App) openProjectDialog() {
	err := os.MkdirAll(projectsBaseDir, 0755)
	if err != nil {
		dialog.ShowError(fmt.Errorf("プロジェクトディレクトリの確認/作成に失敗しました: %w", err), a.window)
		return
	}

	entries, err := os.ReadDir(projectsBaseDir)
	if err != nil {
		dialog.ShowError(fmt.Errorf("プロジェクトの読み込みに失敗しました: %w", err), a.window)
		return
	}

	var projectIDs []string
	var projectDisplayNames []string

	for _, entry := range entries {
		if entry.IsDir() {
			projectID := entry.Name()
			yamlFile := filepath.Join(projectsBaseDir, projectID, yamlFileName)
			yamlData, readErr := ioutil.ReadFile(yamlFile)
			displayName := projectID
			if readErr == nil {
				var tree TreeData
				if unmarshalErr := yaml.Unmarshal(yamlData, &tree); unmarshalErr == nil && tree.ProjectName != "" {
					displayName = fmt.Sprintf("%s (%s)", tree.ProjectName, projectID)
				} else if unmarshalErr != nil {
					log.Printf("Error unmarshalling project %s's tree.yaml for name: %v", projectID, unmarshalErr)
				}
			} else {
				log.Printf("Error reading project %s's tree.yaml for name: %v", projectID, readErr)
			}
			projectIDs = append(projectIDs, projectID)
			projectDisplayNames = append(projectDisplayNames, displayName)
		}
	}

	if len(projectIDs) == 0 {
		dialog.ShowInformation("プロジェクトを開く", "保存されているプロジェクトはありません。", a.window)
		return
	}

	var selectedProjectID string

	projectList := widget.NewList(
		func() int {
			return len(projectDisplayNames)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(projectDisplayNames[i])
		},
	)
	projectList.OnSelected = func(id widget.ListItemID) {
		selectedProjectID = projectIDs[id]
		log.Printf("Project selected in list: %s (ID: %s)", projectDisplayNames[id], selectedProjectID)
	}

	listContainer := container.NewVScroll(projectList)
	listContainer.SetMinSize(fyne.NewSize(300, 200))

	dialog.ShowCustomConfirm("プロジェクトを開く", "開く", "キャンセル", listContainer, func(confirm bool) {
		if confirm && selectedProjectID != "" {
			log.Printf("Confirmed to load project ID: %s", selectedProjectID)
			fyne.Do(func() {
				a.loadProjectData(selectedProjectID)
			})
		} else if confirm && selectedProjectID == "" {
			dialog.ShowInformation("情報", "プロジェクトが選択されていません。", a.window)
		}
	}, a.window)
}

func (a *App) saveCurrentProject() {
	if a.currentProjectID == "" {
		log.Println("saveCurrentProject: No active project to save.")
		return
	}
	saveData(a, a.currentProjectID, a.currentProjectName)
}

func (a *App) loadProjectData(projectID string) {
	loadProjectData(a, projectID)
}

func (a *App) clearCurrentProjectState() {
	log.Println("Clearing current project state.")
	a.nodesMutex.Lock()
	a.nodes = []*ui.NodeData{}
	a.nodesMutex.Unlock()

	if a.dialogCanvas != nil {
		a.dialogCanvas.Clear()
		// a.dialogCanvas.selectedBranchSourceID = "" // This line is no longer needed
	}
	a.currentProjectID = ""
	a.currentProjectName = ""
	a.updateWindowTitle()
	if a.statusLabel != nil {
		a.statusLabel.SetText("準備完了 (プロジェクトなし)")
	}
	if a.dialogCanvas != nil {
		a.dialogCanvas.Refresh()
	}
}

// saveData は指定されたプロジェクトIDと名前で現在のノードデータを保存します。
func saveData(appInstance *App, projectID string, projectName string) {
	if projectID == "" {
		log.Println("saveData: projectID is empty, cannot save.")
		return
	}
	log.Printf("Saving project ID: %s, Name: %s", projectID, projectName)

	appInstance.nodesMutex.RLock()
	nodesToSave := make([]*ui.NodeData, len(appInstance.nodes))
	for i, n := range appInstance.nodes {
		nodeCopy := *n
		nodesToSave[i] = &nodeCopy
	}
	appInstance.nodesMutex.RUnlock()

	tree := TreeData{Nodes: nodesToSave, ProjectName: projectName}
	projectDataPath := filepath.Join(projectsBaseDir, projectID)
	yamlFile := filepath.Join(projectDataPath, yamlFileName)
	mdDir := filepath.Join(projectDataPath, mdNodesDirName)

	if err := os.MkdirAll(projectsBaseDir, 0755); err != nil {
		log.Printf("ベースプロジェクトディレクトリ作成エラー (%s): %v", projectsBaseDir, err)
		if appInstance.window != nil {
			dialog.ShowError(fmt.Errorf("プロジェクトディレクトリの作成に失敗しました: %w", err), appInstance.window)
		}
		return
	}
	if err := os.MkdirAll(projectDataPath, 0755); err != nil {
		log.Printf("プロジェクトデータディレクトリ作成エラー (%s): %v", projectDataPath, err)
		if appInstance.window != nil {
			dialog.ShowError(fmt.Errorf("プロジェクトデータディレクトリの作成に失敗しました: %w", err), appInstance.window)
		}
		return
	}

	yamlData, err := yaml.Marshal(&tree)
	if err != nil {
		log.Printf("YAMLマーシャリングエラー: %v", err)
		if appInstance.window != nil {
			dialog.ShowError(err, appInstance.window)
		}
		return
	}
	err = ioutil.WriteFile(yamlFile, yamlData, 0644)
	if err != nil {
		log.Printf("YAMLファイル書き込みエラー (%s): %v", yamlFile, err)
		if appInstance.window != nil {
			dialog.ShowError(err, appInstance.window)
		}
		return
	}

	err = os.MkdirAll(mdDir, 0755)
	if err != nil {
		log.Printf("Markdownディレクトリ作成エラー (%s): %v", mdDir, err)
		if appInstance.window != nil {
			dialog.ShowError(err, appInstance.window)
		}
		return
	}

	for _, node := range nodesToSave {
		mdContent := fmt.Sprintf("# Question\n\n%s\n\n---\n\n# Answer\n\n%s", node.Question, node.Answer)
		mdPath := filepath.Join(mdDir, node.ID+".md")
		err = ioutil.WriteFile(mdPath, []byte(mdContent), 0644)
		if err != nil {
			log.Printf("Markdownファイル書き込みエラー (%s): %v", mdPath, err)
		}
	}

	log.Println("データが正常に保存されました。")
	if appInstance.statusLabel != nil {
		appInstance.statusLabel.SetText(fmt.Sprintf("プロジェクト「%s」保存完了", projectName))
	}
}

// loadProjectData は指定されたプロジェクトIDのデータを読み込み、mainAppの状態を更新します。
func loadProjectData(appInstance *App, projectID string) {
	if projectID == "" {
		log.Println("loadProjectData: projectID is empty.")
		appInstance.clearCurrentProjectState()
		appInstance.updateWindowTitle()
		if appInstance.dialogCanvas != nil {
			appInstance.dialogCanvas.Refresh()
		}
		return
	}
	log.Printf("Loading project ID: %s", projectID)
	projectDataPath := filepath.Join(projectsBaseDir, projectID)
	yamlFile := filepath.Join(projectDataPath, yamlFileName)
	mdDir := filepath.Join(projectDataPath, mdNodesDirName)

	yamlData, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("プロジェクトファイルが見つかりません: %s", yamlFile)
			if appInstance.window != nil {
				dialog.ShowError(fmt.Errorf("プロジェクトファイル '%s' が見つかりません。", projectID), appInstance.window)
			}
			appInstance.clearCurrentProjectState()
			return
		}
		log.Printf("YAMLファイル読み込みエラー: %v", err)
		if appInstance.window != nil {
			dialog.ShowError(err, appInstance.window)
		}
		appInstance.clearCurrentProjectState()
		return
	}

	var tree TreeData
	err = yaml.Unmarshal(yamlData, &tree)
	if err != nil {
		log.Printf("YAMLアンマーシャリングエラー: %v", err)
		if appInstance.window != nil {
			dialog.ShowError(err, appInstance.window)
		}
		appInstance.clearCurrentProjectState()
		return
	}

	appInstance.currentProjectID = projectID
	appInstance.currentProjectName = tree.ProjectName
	appInstance.updateWindowTitle()

	loadedNodes := []*ui.NodeData{}
	for _, node := range tree.Nodes {
		mdPath := filepath.Join(mdDir, node.ID+".md")
		q, ans, errMd := parseMarkdown(mdPath)
		if errMd != nil {
			log.Printf("Markdownファイル読み込みエラー (%s): %v", mdPath, errMd)
			q = "(質問読み込みエラー)"
			ans = fmt.Sprintf("Markdownファイル '%s' の読み込みに失敗しました: %v", mdPath, errMd)
		}
		node.Question = q
		node.Answer = ans
		loadedNodes = append(loadedNodes, node)
	}

	appInstance.clearCurrentProjectState() // Clear before loading new nodes, but preserve new project ID/Name

	appInstance.nodesMutex.Lock()
	appInstance.nodes = loadedNodes // Assign loaded nodes to mainApp
	appInstance.nodesMutex.Unlock()

	if appInstance.dialogCanvas != nil {
		for _, nodeData := range appInstance.nodes { // Use appInstance.nodes which is now populated
			appInstance.dialogCanvas.AddNode(nodeData)
		}
	}

	log.Printf("プロジェクト「%s」が正常に読み込まれました。", appInstance.currentProjectName)
	if appInstance.statusLabel != nil {
		appInstance.statusLabel.SetText(fmt.Sprintf("プロジェクト「%s」読み込み完了", appInstance.currentProjectName))
	}
	if appInstance.dialogCanvas != nil {
		appInstance.dialogCanvas.Refresh()
	}
}

func parseMarkdown(filePath string) (string, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var questionLines, answerLines []string
	var readingQuestion, readingAnswer bool

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# Question") {
			readingQuestion = true
			readingAnswer = false
			if len(questionLines) > 0 {
				questionLines = []string{}
			}
			continue
		}
		if strings.HasPrefix(line, "# Answer") {
			readingQuestion = false
			readingAnswer = true
			if len(answerLines) > 0 {
				answerLines = []string{}
			}
			continue
		}
		if strings.HasPrefix(line, "---") {
			if readingQuestion {
				readingQuestion = false
			}
			continue
		}

		if readingQuestion {
			questionLines = append(questionLines, line)
		} else if readingAnswer {
			answerLines = append(answerLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", err
	}

	qStr := strings.TrimSpace(strings.Join(questionLines, "\n"))
	aStr := strings.TrimSpace(strings.Join(answerLines, "\n"))

	return qStr, aStr, nil
}

// Run is the main entry point of the application
func (a *App) Run() {
	go a.handleUIUpdates()
	a.window.Show()
	a.fyneApp.Run()
}
