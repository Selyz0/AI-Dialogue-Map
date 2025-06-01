package ui

import (
	"log"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// DialogCanvas はノードと接続線を描画するカスタムウィジェットです。
type DialogCanvas struct {
	widget.BaseWidget
	nodes                  []*NodeWidget
	nodeMap                map[string]*NodeWidget
	content                *fyne.Container
	lineLayer              *fyne.Container
	selectedBranchSourceID string
	app                    fyne.App
	nodesMutex             sync.RWMutex
	lines                  []*canvas.Line
	linesMutex             sync.RWMutex
	viewOffset             fyne.Position
	zoomFactor             float32
	onNodeDeleted          func(nodeID string)
}

// NewDialogCanvas は新しいDialogCanvasのインスタンスを作成します。
func NewDialogCanvas(app fyne.App, onNodeDeleted func(nodeID string)) *DialogCanvas {
	dc := &DialogCanvas{
		nodes:         make([]*NodeWidget, 0),
		nodeMap:       make(map[string]*NodeWidget),
		app:           app,
		lines:         make([]*canvas.Line, 0),
		zoomFactor:    1.0,
		viewOffset:    fyne.NewPos(0, 0),
		onNodeDeleted: onNodeDeleted,
	}
	dc.ExtendBaseWidget(dc)
	dc.content = container.NewWithoutLayout()
	dc.lineLayer = container.NewWithoutLayout()
	return dc
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (dc *DialogCanvas) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	return &dialogCanvasRenderer{
		canvas:  dc,
		bg:      bg,
		objects: []fyne.CanvasObject{bg, dc.lineLayer, dc.content},
	}
}

// AddNode は新しいノードをキャンバスに追加します。
func (dc *DialogCanvas) AddNode(data *NodeData) {
	log.Printf("DialogCanvas.AddNode START - ID: %s, ParentID: %s, Title: %s", data.ID, data.ParentID, data.Title)
	dc.nodesMutex.Lock()
	defer dc.nodesMutex.Unlock()

	nodeWidget := NewNodeWidget(data, dc)
	nodeWidget.onDragChanged = func() {
		fyne.Do(func() {
			log.Printf("Node %s drag changed, refreshing canvas", nodeWidget.data.ID)
			dc.Refresh()
		})
	}
	nodeWidget.onBranchRequested = func(d *NodeData) {
		fyne.Do(func() {
			dc.SetBranchSource(d.ID)
		})
	}
	nodeWidget.onDeleteRequested = dc.onNodeDeleted

	if data.ParentID != "" {
		parent := dc.nodeMap[data.ParentID]
		if parent != nil {
			parentModelPos := parent.data.Position
			parentModelSize := parent.MinSize()
			newNodeModelSize := nodeWidget.MinSize()

			childrenCount := 0
			for _, n := range dc.nodes {
				if n.data.ParentID == data.ParentID {
					childrenCount++
				}
			}

			newX := parentModelPos.X + parentModelSize.Width + nodeSpacing
			newY := parentModelPos.Y + (float32(childrenCount) * (newNodeModelSize.Height + nodeSpacing/2)) - parentModelSize.Height/2 + newNodeModelSize.Height/2

			calculatedPosition := fyne.NewPos(newX, newY)

			for _, existingNode := range dc.nodes {
				if existingNode.data.ID != data.ID {
					existingPos := existingNode.data.Position
					existingSize := existingNode.MinSize()
					overlapX := calculatedPosition.X < existingPos.X+existingSize.Width && calculatedPosition.X+newNodeModelSize.Width > existingPos.X
					overlapY := calculatedPosition.Y < existingPos.Y+existingSize.Height && calculatedPosition.Y+newNodeModelSize.Height > existingPos.Y
					if overlapX && overlapY {
						calculatedPosition.X += nodeSpacing
						calculatedPosition.Y += nodeSpacing
						break
					}
				}
			}
			data.Position = calculatedPosition
		} else {
			l := len(dc.nodes)
			data.Position = fyne.NewPos(float32(l%5*int(nodeWidthCollapsed+20)+50), float32(l/5*int(nodeHeightCollapsed+20)+200))
		}
	} else {
		l := len(dc.nodes)
		data.Position = fyne.NewPos(float32(l%5*int(nodeWidthCollapsed+20)+50), float32(l/5*int(nodeHeightCollapsed+20)+200))
	}

	dc.nodes = append(dc.nodes, nodeWidget)
	dc.nodeMap[data.ID] = nodeWidget
	dc.content.Add(nodeWidget)

	log.Printf("DialogCanvas.AddNode END - ID: %s, ModelPos: %v", data.ID, data.Position)
}

// RemoveNodeAndDescendants removes the node with the given ID and all its descendants.
// It returns a slice of IDs of all nodes that were actually removed.
func (dc *DialogCanvas) RemoveNodeAndDescendants(nodeID string) []string {
	log.Printf("DialogCanvas.RemoveNodeAndDescendants START - ID: %s", nodeID)

	nodesToDeleteIDs := make(map[string]bool)
	var collectDescendants func(currentID string)

	dc.nodesMutex.RLock()
	collectDescendants = func(currentID string) {
		if nodesToDeleteIDs[currentID] {
			return
		}
		nodesToDeleteIDs[currentID] = true
		for _, nWidget := range dc.nodes {
			if nWidget.data.ParentID == currentID {
				collectDescendants(nWidget.data.ID)
			}
		}
	}
	collectDescendants(nodeID)
	dc.nodesMutex.RUnlock()

	log.Printf("Nodes to delete (IDs): %v", nodesToDeleteIDs)
	if len(nodesToDeleteIDs) == 0 {
		log.Println("No nodes found to delete.")
		return []string{}
	}

	var actuallyDeletedIDs []string

	dc.nodesMutex.Lock()
	defer dc.nodesMutex.Unlock()

	newNodes := []*NodeWidget{}
	for _, nw := range dc.nodes {
		if !nodesToDeleteIDs[nw.data.ID] {
			newNodes = append(newNodes, nw)
		} else {
			dc.content.Remove(nw)
			delete(dc.nodeMap, nw.data.ID)
			actuallyDeletedIDs = append(actuallyDeletedIDs, nw.data.ID)
			log.Printf("Node %s (widget and map entry) removed from DialogCanvas.", nw.data.ID)
		}
	}
	dc.nodes = newNodes

	log.Printf("DialogCanvas.RemoveNodeAndDescendants END, deleted count: %d", len(actuallyDeletedIDs))
	return actuallyDeletedIDs
}

func (dc *DialogCanvas) findNodeWidgetByID(id string) *NodeWidget {
	dc.nodesMutex.RLock()
	defer dc.nodesMutex.RUnlock()
	return dc.nodeMap[id]
}

func (dc *DialogCanvas) updateNodeConnections() {
	log.Println("DialogCanvas.updateNodeConnections START")
	dc.linesMutex.Lock()
	defer dc.linesMutex.Unlock()
	dc.nodesMutex.RLock()
	defer dc.nodesMutex.RUnlock()

	for _, line := range dc.lines {
		dc.lineLayer.Remove(line)
	}
	dc.lines = []*canvas.Line{}

	for _, childNode := range dc.nodes {
		if childNode == nil || childNode.data == nil {
			continue
		}
		if childNode.data.ParentID != "" {
			parentNode := dc.nodeMap[childNode.data.ParentID]

			if parentNode != nil && parentNode.data != nil {
				line := canvas.NewLine(theme.Color(theme.ColorNameForeground))
				line.StrokeWidth = 1.5

				parentScreenPos := parentNode.Position()
				parentScreenSize := parentNode.Size()
				childScreenPos := childNode.Position()
				childScreenSize := childNode.Size()

				line.Position1 = fyne.NewPos(parentScreenPos.X+parentScreenSize.Width, parentScreenPos.Y+parentScreenSize.Height/2)
				line.Position2 = fyne.NewPos(childScreenPos.X, childScreenPos.Y+childScreenSize.Height/2)

				dc.lines = append(dc.lines, line)
				dc.lineLayer.Add(line)
			}
		}
	}
	dc.lineLayer.Refresh()
	log.Println("DialogCanvas.updateNodeConnections END")
}

func (dc *DialogCanvas) clearConnections() {
	dc.linesMutex.Lock()
	defer dc.linesMutex.Unlock()
	for _, line := range dc.lines {
		dc.lineLayer.Remove(line)
	}
	dc.lines = []*canvas.Line{}
	dc.lineLayer.Refresh()
}

func (dc *DialogCanvas) SetBranchSource(nodeID string) {
	dc.nodesMutex.Lock()
	defer dc.nodesMutex.Unlock()

	previousSourceID := dc.selectedBranchSourceID
	dc.selectedBranchSourceID = nodeID
	log.Printf("分岐元が %s に設定されました。", nodeID)

	for _, nw := range dc.nodes {
		isNowSource := nw.data.ID == nodeID
		if nw.data.IsBranchSource != isNowSource || (nw.data.ID == previousSourceID && previousSourceID != nodeID) {
			nw.data.IsBranchSource = isNowSource
			nw.Refresh()
		}
	}
	// dc.Refresh() // This might be redundant if individual node refreshes are enough
}

func (dc *DialogCanvas) GetBranchSource() string {
	dc.nodesMutex.RLock()
	defer dc.nodesMutex.RUnlock()
	return dc.selectedBranchSourceID
}

func (dc *DialogCanvas) Dragged(e *fyne.DragEvent) {
	dc.viewOffset = dc.viewOffset.Add(fyne.NewPos(e.Dragged.DX, e.Dragged.DY))
	log.Printf("Pan: Offset: %v", dc.viewOffset)
	dc.Refresh()
}
func (dc *DialogCanvas) DragEnd() {}

func (dc *DialogCanvas) Scrolled(ev *fyne.ScrollEvent) { // Zooming
	currentDriver := fyne.CurrentApp().Driver()
	if deskDrv, ok := currentDriver.(desktop.Driver); ok {
		if (deskDrv.CurrentKeyModifiers() & fyne.KeyModifierControl) == 0 {
			return
		}
	} else if fyne.CurrentApp().Driver().Device().IsMobile() {
	} else {
		return
	}

	canvasSize := dc.Size()
	canvasCenterInView := fyne.NewPos(canvasSize.Width/2, canvasSize.Height/2)

	oldZoom := dc.zoomFactor

	if ev.Scrolled.DY > 0 {
		dc.zoomFactor *= 1.1
	} else if ev.Scrolled.DY < 0 {
		dc.zoomFactor /= 1.1
	}
	if dc.zoomFactor < 0.1 {
		dc.zoomFactor = 0.1
	}
	if dc.zoomFactor > 5.0 {
		dc.zoomFactor = 5.0
	}

	modelX := (canvasCenterInView.X - dc.viewOffset.X) / oldZoom
	modelY := (canvasCenterInView.Y - dc.viewOffset.Y) / oldZoom

	dc.viewOffset.X = canvasCenterInView.X - (modelX * dc.zoomFactor)
	dc.viewOffset.Y = canvasCenterInView.Y - (modelY * dc.zoomFactor)

	log.Printf("Zoom: %f, Offset: %v", dc.zoomFactor, dc.viewOffset)
	dc.Refresh()
}

func (dc *DialogCanvas) Clear() {
	dc.nodesMutex.Lock()
	defer dc.nodesMutex.Unlock()
	for _, n := range dc.nodes {
		dc.content.Remove(n)
	}
	dc.nodes = []*NodeWidget{}
	dc.nodeMap = make(map[string]*NodeWidget) // Clear the map
	dc.clearConnections()
	dc.content.Refresh()
}

func (dc *DialogCanvas) GetNodesData() []*NodeData {
	dc.nodesMutex.RLock()
	defer dc.nodesMutex.RUnlock()
	data := make([]*NodeData, len(dc.nodes))
	for i, nw := range dc.nodes {
		data[i] = nw.data
	}
	return data
}

type dialogCanvasRenderer struct {
	canvas  *DialogCanvas
	bg      *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *dialogCanvasRenderer) Destroy() {}
func (r *dialogCanvasRenderer) Layout(size fyne.Size) {
	log.Println("DialogCanvasRenderer.Layout START, size:", size)
	r.bg.Resize(size)
	r.canvas.lineLayer.Resize(size)
	r.canvas.lineLayer.Move(fyne.NewPos(0, 0))
	r.canvas.content.Resize(size)
	r.canvas.content.Move(fyne.NewPos(0, 0))

	r.canvas.nodesMutex.RLock()
	nodesCopy := make([]*NodeWidget, len(r.canvas.nodes))
	copy(nodesCopy, r.canvas.nodes)
	r.canvas.nodesMutex.RUnlock()

	for _, nw := range nodesCopy {
		if nw == nil || nw.data == nil {
			log.Printf("DialogCanvasRenderer.Layout: Encountered nil NodeWidget or NodeData, skipping.")
			continue
		}
		screenPos := fyne.NewPos(nw.data.Position.X*r.canvas.zoomFactor, nw.data.Position.Y*r.canvas.zoomFactor).Add(r.canvas.viewOffset)
		modelMinSize := nw.MinSize()
		screenSize := fyne.NewSize(modelMinSize.Width*r.canvas.zoomFactor, modelMinSize.Height*r.canvas.zoomFactor)

		nw.Move(screenPos)
		nw.Resize(screenSize)
		nw.Refresh()
	}
	// r.canvas.updateNodeConnections() // Moved to Refresh to avoid potential loops
	log.Println("DialogCanvasRenderer.Layout END")
}
func (r *dialogCanvasRenderer) MinSize() fyne.Size           { return fyne.NewSize(300, 200) }
func (r *dialogCanvasRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *dialogCanvasRenderer) Refresh() {
	log.Println("DialogCanvasRenderer.Refresh START for canvas:", r.canvas)
	r.bg.FillColor = theme.Color(theme.ColorNameBackground)
	r.bg.Refresh()

	r.Layout(r.canvas.Size())

	r.canvas.content.Refresh()
	r.canvas.updateNodeConnections()
	// r.canvas.lineLayer.Refresh() // updateNodeConnections already refreshes lineLayer
	log.Println("DialogCanvasRenderer.Refresh END for canvas:", r.canvas)
}
