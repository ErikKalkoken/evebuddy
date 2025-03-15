package fynetree

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Tree is a simplified tree widget for the Fyne GUI toolkit.
type Tree[T TreeNode] struct {
	widget.BaseWidget

	OnSelected func(n T)

	data *TreeData[T]
	tree *widget.Tree
}

func NewTree[T TreeNode](
	create func(b bool) fyne.CanvasObject,
	update func(n T, b bool, co fyne.CanvasObject),
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
	w.data = NewTreeData[T]()
	w.Refresh()
}

func (w *Tree[T]) Set(data *TreeData[T]) {
	w.data = data
	w.Refresh()
}

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
