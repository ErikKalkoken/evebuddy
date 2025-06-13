package widget

import (
	"errors"
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Tree is a simpler to use tree widget for the Fyne GUI toolkit.
//
// The main difference is that it's nodes are defined with the [TreeNodes] API.
type Tree[T TreeNode] struct {
	widget.BaseWidget

	OnSelected func(n T)

	data TreeNodes[T]
	tree *widget.Tree
}

// NewTree returns a new [Tree] object.
func NewTree[T TreeNode](
	create func(isBranch bool) fyne.CanvasObject,
	update func(n T, isBranch bool, co fyne.CanvasObject),
) *Tree[T] {
	w := &Tree[T]{
		data: TreeNodes[T]{},
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
	w.data.Clear()
	w.Refresh()
}

// Set updates all nodes of a tree.
func (w *Tree[T]) Set(data TreeNodes[T]) {
	w.data = data
	w.Refresh()
}

// Data returns the data for a tree.
func (w *Tree[T]) Data() TreeNodes[T] {
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

const (
	RootUID widget.TreeNodeID = ""
)

var ErrUndefined = errors.New("object undefined")

// TreeNode is the interface for a node in a Fyne tree.
type TreeNode interface {
	// UID returns a unique ID for a node.
	UID() widget.TreeNodeID
}

// TreeNodes holds all the nodes for rendering a [Tree] widget.
//
// It is designed to make it easier to construct tree data structures for a widget
// by hiding some of the complexity behind a simpler API and by performing sanity checks.
//
// Trees are constructed by adding nodes, which can contain any data
// as long as it complies with the [TreeNode] interface.
//
// Please note that nodes that have child nodes are always reported as branches.
// This means there can not be any empty branch nodes.
//
// The zero value is an empty tree ready to use.
// This type is not thread safe.
type TreeNodes[T TreeNode] struct {
	ids     map[widget.TreeNodeID][]widget.TreeNodeID
	parents map[widget.TreeNodeID]widget.TreeNodeID
	nodes   map[widget.TreeNodeID]T
}

// Add adds a node safely. It returns it's UID or an error if the node can not be added.
//
// Use [RootUID] as parentUID for adding nodes at the top level.
// Nodes will be rendered in the same order as they are added.
func (t *TreeNodes[T]) Add(parentUID widget.TreeNodeID, node T) (widget.TreeNodeID, error) {
	if t == nil {
		return "", ErrUndefined
	}
	if t.ids == nil || t.parents == nil || t.nodes == nil {
		t.init()
	}
	if parentUID != "" {
		_, found := t.nodes[parentUID]
		if !found {
			return "", fmt.Errorf("parent node does not exist: %s", parentUID)
		}
	}
	uid := node.UID()
	if uid == "" {
		return "", fmt.Errorf("UID() must not return zero value: %+v", node)
	}
	_, found := t.nodes[uid]
	if found {
		return "", fmt.Errorf("this node already exists: %+v", node)
	}
	t.ids[parentUID] = append(t.ids[parentUID], uid)
	t.nodes[uid] = node
	t.parents[uid] = parentUID
	return uid, nil
}

func (t *TreeNodes[T]) init() {
	t.ids = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	t.parents = make(map[widget.TreeNodeID]widget.TreeNodeID)
	t.nodes = make(map[widget.TreeNodeID]T)
}

// Clear removes all nodes.
func (t *TreeNodes[T]) Clear() {
	if t == nil {
		panic(ErrUndefined)
	}
	t.init()
}

// ChildUIDs returns the child UIDs of a node.
func (t TreeNodes[T]) ChildUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	return t.ids[uid]
}

// FIXME: Method does not return full tree
func (t TreeNodes[T]) Flat() []T {
	var s []T
	uid := ""
	for _, id := range t.ChildUIDs(uid) {
		s = append(s, t.MustNode(id))
	}
	return s
}

// IsBranch reports whether a node is a branch.
func (t TreeNodes[T]) IsBranch(uid widget.TreeNodeID) bool {
	_, found := t.ids[uid]
	return found
}

// MustAdd adds a node or panics if the node can not be added.
func (t *TreeNodes[T]) MustAdd(parentUID widget.TreeNodeID, node T) widget.TreeNodeID {
	uid, err := t.Add(parentUID, node)
	if err != nil {
		panic(err)
	}
	return uid
}

// MustNode returns a node or panics if the node does not exist.
// This method mainly exists to simplify test code and should not be used in production code.
func (t TreeNodes[T]) MustNode(uid widget.TreeNodeID) T {
	v, ok := t.Node(uid)
	if !ok {
		panic(fmt.Sprintf("node %s does not exist", uid))
	}
	return v
}

// Node returns a node by UID and reports whether it exists.
//
// Note that when using this method with a Fyne widget it is possible,
// that a UID forwarded by the widget no longer exists due to race conditions.
// It is therefore recommended to always check the ok value.
func (t TreeNodes[T]) Node(uid widget.TreeNodeID) (node T, ok bool) {
	node, ok = t.node(uid)
	return
}

// node returns a node by UID and reports whether it exists.
func (t TreeNodes[T]) node(uid widget.TreeNodeID) (T, bool) {
	v, ok := t.nodes[uid]
	return v, ok
}

// Path returns the UIDs of nodes between a given node and the root.
func (t TreeNodes[T]) Path(uid widget.TreeNodeID) []widget.TreeNodeID {
	path := make([]widget.TreeNodeID, 0)
	for {
		uid = t.parents[uid]
		if uid == "" {
			break
		}
		path = append(path, uid)
	}
	slices.Reverse(path)
	return path
}

// Parent returns the UID of the parent node.
func (t TreeNodes[T]) Parent(uid widget.TreeNodeID) (parent widget.TreeNodeID, ok bool) {
	parent, ok = t.parents[uid]
	return
}

// Size returns the number of nodes in the tree
func (t TreeNodes[T]) Size() int {
	return len(t.nodes)
}
