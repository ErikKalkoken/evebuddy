package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
)

// Tree is a simplified tree widget that is defined with a Fynetree.
type Tree[T fynetree.TreeNode] struct {
	widget.BaseWidget

	OnSelected func(n T)

	data *fynetree.FyneTree[T]
	tree *widget.Tree
}

func NewTree[T fynetree.TreeNode](
	create func(b bool) fyne.CanvasObject,
	update func(n T, b bool, co fyne.CanvasObject),
) *Tree[T] {
	w := &Tree[T]{
		data: fynetree.New[T](),
	}
	w.tree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return w.data.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return w.data.IsBranch(uid)
		},
		create,
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			n, ok := w.data.Node(uid)
			if !ok {
				return
			}
			update(n, b, co)
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

func (w *Tree[T]) Clear() {
	w.data = fynetree.New[T]()
	w.Refresh()
}

func (w *Tree[T]) Set(data *fynetree.FyneTree[T]) {
	w.data = data
	w.Refresh()
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
