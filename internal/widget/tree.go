package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Tree is a simpler to use tree widget for the Fyne GUI toolkit.
//
// The main difference is that it's data is defined through the [TreeData] API.
type Tree[T TreeNode] struct {
	widget.BaseWidget

	OnSelected func(n T)

	data *TreeData[T]
	tree *widget.Tree
}

// NewTree returns a new [Tree] object.
func NewTree[T TreeNode](
	create func(isBranch bool) fyne.CanvasObject,
	update func(n T, isBranch bool, co fyne.CanvasObject),
) *Tree[T] {
	w := &Tree[T]{
		data: NewTreeData[T](),
	}
	w.tree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return w.data.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return w.data.IsBranch(uid)
		},
		create,
		func(uid widget.TreeNodeID, isBranch bool, co fyne.CanvasObject) {
			n, ok := w.data.Node(uid)
			if !ok {
				return
			}
			update(n, isBranch, co)
		},
	)
	w.tree.OnSelected = func(uid widget.TreeNodeID) {
		if w.OnSelected != nil {
			n, ok := w.data.Node(uid)
			if !ok {
				return
			}
			w.OnSelected(n)
		}
	}
	w.ExtendBaseWidget(w)
	return w
}

// Clear removes all nodes from the tree.
func (w *Tree[T]) Clear() {
	w.data = NewTreeData[T]()
	w.Refresh()
}

// Set updates the all nodes of a tree.
func (w *Tree[T]) Set(data *TreeData[T]) {
	w.data = data
	w.Refresh()
}

// Data returns the tree data for a tree.
func (w *Tree[T]) Data() *TreeData[T] {
	return w.data
}

func (w *Tree[T]) OpenAllBranches() {
	w.tree.OpenAllBranches()
}

func (w *Tree[T]) CloseAllBranches() {
	w.tree.CloseAllBranches()
}

func (w *Tree[T]) OpenBranch(n T) {
	w.tree.OpenBranch(n.UID())
}

func (w *Tree[T]) CloseBranch(n T) {
	w.tree.CloseBranch(n.UID())
}

func (w *Tree[T]) ToggleBranch(n T) {
	w.tree.ToggleBranch(n.UID())
}

func (w *Tree[T]) ScrollToTop() {
	w.tree.ScrollToTop()
}

func (w *Tree[T]) ScrollTo(n T) {
	w.tree.ScrollTo(n.UID())
}

func (w *Tree[T]) Select(n T) {
	w.tree.Select(n.UID())
}

func (w *Tree[T]) UnselectAll() {
	w.tree.UnselectAll()
}

func (w *Tree[T]) Refresh() {
	w.tree.Refresh()
	w.BaseWidget.Refresh()
}

func (w *Tree[T]) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.tree)
}
