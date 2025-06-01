package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"AI-Dialogue-Map/internal/utils"
)

const (
	nodeSpacing             float32 = 40
	nodeWidthCollapsed      float32 = 220
	nodeHeightCollapsed     float32 = 110
	nodeWidthExpanded       float32 = 380
	maxNodeHeightExpanded   float32 = 1200
	maxAnswerLinesCollapsed         = 20
	nodeTitleMaxLength              = 25
)

// NodeData はノードのデータを保持します。
// This struct is now defined here and used by other files in the 'main' package.
type NodeData struct {
	ID             string        `yaml:"id"`
	Title          string        `yaml:"title"`
	Question       string        `yaml:"-"`
	Answer         string        `yaml:"-"`
	Position       fyne.Position `yaml:"position"`
	Expanded       bool          `yaml:"expanded"`
	ParentID       string        `yaml:"parent_id,omitempty"`
	IsBranchSource bool          `yaml:"-"`
}

// NodeWidget はキャンバス上の単一ノードを表すウィジェットです。
// This struct is now defined here.
type NodeWidget struct {
	widget.BaseWidget
	data              *NodeData
	rect              *canvas.Rectangle
	titleLabel        *widget.Label
	answerDisplay     *widget.RichText
	answerScroll      *container.Scroll
	expandButton      *widget.Button
	branchButton      *widget.Button
	deleteButton      *widget.Button
	onDragChanged     func()
	onBranchRequested func(*NodeData)
	onDeleteRequested func(nodeID string)
	dialogCanvas      *DialogCanvas // Reference to the parent canvas (DialogCanvas defined in dialog_canvas.go)
}

// NewNodeWidget は新しいNodeWidgetのインスタンスを作成します。
func NewNodeWidget(data *NodeData, canvas *DialogCanvas) *NodeWidget {
	nw := &NodeWidget{
		data:         data,
		dialogCanvas: canvas,
	}
	nw.ExtendBaseWidget(nw)
	return nw
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (nw *NodeWidget) CreateRenderer() fyne.WidgetRenderer {
	nw.rect = canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	nw.rect.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	nw.rect.StrokeWidth = 1
	nw.rect.CornerRadius = theme.Size(theme.SizeNamePadding)

	nw.titleLabel = widget.NewLabel(utils.TruncateText(nw.data.Title, nodeTitleMaxLength))
	nw.titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	nw.titleLabel.Wrapping = fyne.TextWrapWord

	nw.answerDisplay = widget.NewRichTextFromMarkdown("")
	nw.answerDisplay.Wrapping = fyne.TextWrapWord
	nw.answerScroll = container.NewScroll(nw.answerDisplay)

	nw.expandButton = widget.NewButtonWithIcon("", theme.MoreVerticalIcon(), func() {
		nw.data.Expanded = !nw.data.Expanded
		nw.Refresh()
		if nw.dialogCanvas != nil {
			nw.dialogCanvas.Refresh()
		}
	})

	nw.branchButton = widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		if nw.onBranchRequested != nil {
			nw.onBranchRequested(nw.data)
		}
	})

	nw.deleteButton = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		var topWindow fyne.Window
		currentApp := fyne.CurrentApp()
		if currentApp != nil {
			windows := currentApp.Driver().AllWindows()
			if len(windows) > 0 {
				topWindow = windows[0]
			}
		}
		if topWindow == nil {
			log.Println("警告: 削除確認ダイアログの表示ウィンドウが見つかりません。")
			if nw.onDeleteRequested != nil {
				nw.onDeleteRequested(nw.data.ID)
			}
			return
		}

		dialog.ShowConfirm("ノード削除", fmt.Sprintf("ノード「%s」を削除してもよろしいですか？\n(この操作は元に戻せません)", nw.data.Title), func(confirm bool) {
			if confirm {
				if nw.onDeleteRequested != nil {
					nw.onDeleteRequested(nw.data.ID)
				}
			}
		}, topWindow)
	})
	nw.deleteButton.Importance = widget.LowImportance

	nw.expandButton.Importance = widget.LowImportance
	nw.branchButton.Importance = widget.LowImportance

	titleBar := container.NewBorder(nil, nil, nil, nw.deleteButton, nw.titleLabel)

	mainContentArea := container.NewBorder(
		titleBar,
		container.NewHBox(layout.NewSpacer(), nw.expandButton),
		nil,
		nil,
		nw.answerScroll,
	)

	branchButtonContainer := container.NewVBox(
		layout.NewSpacer(),
		nw.branchButton,
		layout.NewSpacer(),
	)

	contentBox := container.NewBorder(
		nil, nil, nil, branchButtonContainer,
		mainContentArea,
	)

	objects := []fyne.CanvasObject{nw.rect, contentBox}

	r := &nodeWidgetRenderer{
		widget:     nw,
		objects:    objects,
		rect:       nw.rect,
		contentBox: contentBox,
	}
	r.Refresh()
	return r
}

// Dragged is called when a drag event occurs on the widget.
func (nw *NodeWidget) Dragged(e *fyne.DragEvent) {
	if nw.dialogCanvas != nil && nw.dialogCanvas.zoomFactor != 0 {
		modelDelta := fyne.NewPos(e.Dragged.DX/nw.dialogCanvas.zoomFactor, e.Dragged.DY/nw.dialogCanvas.zoomFactor)
		nw.data.Position = nw.data.Position.Add(modelDelta)
	} else {
		modelDelta := fyne.NewPos(e.Dragged.DX, e.Dragged.DY)
		nw.data.Position = nw.data.Position.Add(modelDelta)
	}
	if nw.onDragChanged != nil {
		nw.onDragChanged()
	}
	nw.Refresh()
	if nw.dialogCanvas != nil {
		nw.dialogCanvas.Refresh()
	}
}

// DragEnd is called when a drag event ends.
func (nw *NodeWidget) DragEnd() {
	if nw.onDragChanged != nil {
		nw.onDragChanged()
	}
	if nw.dialogCanvas != nil {
		nw.dialogCanvas.Refresh()
	}
}

// MinSize returns the minimum size of the widget.
func (nw *NodeWidget) MinSize() fyne.Size {
	var targetWidth, targetHeight float32

	if nw.data == nil {
		log.Println("NodeWidget.MinSize: data is nil, returning default.")
		return fyne.NewSize(nodeWidthCollapsed, nodeHeightCollapsed)
	}

	padding := theme.Size(theme.SizeNamePadding)
	textStyle := fyne.TextStyle{Bold: true}
	defaultTextSize := theme.TextSize()
	defaultIconSize := theme.Size(theme.SizeNameInlineIcon)

	var titleTextHeight float32
	if nw.titleLabel != nil {
		titleTextHeight = nw.titleLabel.MinSize().Height
	} else {
		titleTextHeight = fyne.MeasureText("Title", defaultTextSize, textStyle).Height
		if titleTextHeight == 0 {
			titleTextHeight = defaultTextSize * 1.5
		}
	}
	if titleTextHeight < 20 {
		titleTextHeight = 20
	}

	expandButtonHeight := defaultIconSize + padding*2
	branchButtonWidth := defaultIconSize + padding*2
	deleteButtonWidth := defaultIconSize + padding*2

	var answerContentHeight float32
	if nw.data.Expanded {
		answerContentHeight = defaultTextSize * 6
		if answerContentHeight < defaultTextSize*float32(maxAnswerLinesCollapsed+1) {
			answerContentHeight = defaultTextSize*float32(maxAnswerLinesCollapsed+1) + padding
		}
		maxAnswerAreaHeight := maxNodeHeightExpanded - titleTextHeight - expandButtonHeight - padding*4
		if maxAnswerAreaHeight < defaultTextSize*2 {
			maxAnswerAreaHeight = defaultTextSize * 2
		}
		if answerContentHeight > maxAnswerAreaHeight {
			answerContentHeight = maxAnswerAreaHeight
		}
	} else {
		answerContentHeight = defaultTextSize * 1.2 * float32(maxAnswerLinesCollapsed)
		if answerContentHeight > 120 {
			answerContentHeight = 120
		}
		if answerContentHeight < 20 {
			answerContentHeight = 20
		}
		log.Println(answerContentHeight)
	}

	if nw.data.Expanded {
		targetWidth = nodeWidthExpanded + branchButtonWidth + deleteButtonWidth + padding*2
		targetHeight = titleTextHeight + answerContentHeight + expandButtonHeight + padding*4
		if maxNodeHeightExpanded > 0 && targetHeight > maxNodeHeightExpanded {
			targetHeight = maxNodeHeightExpanded
		}
		if targetHeight < nodeHeightCollapsed {
			targetHeight = nodeHeightCollapsed
		}
	} else {
		targetWidth = nodeWidthCollapsed + branchButtonWidth + deleteButtonWidth + padding*2
		targetHeight = nodeHeightCollapsed
	}
	return fyne.NewSize(targetWidth, targetHeight)
}

type nodeWidgetRenderer struct {
	widget     *NodeWidget
	objects    []fyne.CanvasObject
	rect       *canvas.Rectangle
	contentBox *fyne.Container
}

func (r *nodeWidgetRenderer) Destroy() {}

func (r *nodeWidgetRenderer) Layout(size fyne.Size) {
	r.rect.Resize(size)
	padding := theme.Size(theme.SizeNamePadding)

	contentSize := fyne.NewSize(size.Width-(2*padding), size.Height-(2*padding))
	if contentSize.Width < 0 {
		contentSize.Width = 0
	}
	if contentSize.Height < 0 {
		contentSize.Height = 0
	}

	r.contentBox.Resize(contentSize)
	r.contentBox.Move(fyne.NewPos(padding, padding))
}

func (r *nodeWidgetRenderer) MinSize() fyne.Size {
	modelMinSize := r.widget.MinSize()
	if r.widget.dialogCanvas != nil && r.widget.dialogCanvas.zoomFactor != 0 {
		return fyne.NewSize(modelMinSize.Width*r.widget.dialogCanvas.zoomFactor, modelMinSize.Height*r.widget.dialogCanvas.zoomFactor)
	}
	return modelMinSize
}

func (r *nodeWidgetRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *nodeWidgetRenderer) Refresh() {
	r.rect.FillColor = theme.Color(theme.ColorNameInputBackground)
	r.rect.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	r.rect.CornerRadius = theme.Size(theme.SizeNamePadding)
	r.updateAppearance()
	// canvas.Refresh(r.widget) // This is called by the widget's Refresh method if needed
}

func (r *nodeWidgetRenderer) updateAppearance() {
	if r.widget.data.IsBranchSource {
		r.rect.StrokeColor = theme.Color(theme.ColorNamePrimary)
		r.rect.StrokeWidth = 2
	} else {
		r.rect.StrokeColor = theme.Color(theme.ColorNameInputBorder)
		r.rect.StrokeWidth = 1
	}
	r.widget.titleLabel.SetText(utils.TruncateText(r.widget.data.Title, nodeTitleMaxLength))
	if r.widget.data.Expanded {
		r.widget.answerDisplay.ParseMarkdown(r.widget.data.Answer)
		r.widget.expandButton.SetIcon(theme.MenuExpandIcon())
	} else {
		r.widget.answerDisplay.ParseMarkdown(utils.TruncateTextWithEllipsis(r.widget.data.Answer, 200, maxAnswerLinesCollapsed))
		r.widget.expandButton.SetIcon(theme.MoreVerticalIcon())
	}
	r.rect.Refresh()
	r.widget.titleLabel.Refresh()
	r.widget.answerDisplay.Refresh()
	r.widget.answerScroll.Refresh()
	r.widget.expandButton.Refresh()
	r.widget.deleteButton.Refresh()
}
