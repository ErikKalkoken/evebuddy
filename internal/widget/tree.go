package widget

import (
	"errors"
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

const (
	RootUID widget.TreeNodeID = ""
)

var ErrUndefined = errors.New("object undefined")

// TreeNode is the interface for a node in a Fyne tree.
type TreeNode interface {
	// UID returns a unique ID for a node.
	UID() widget.TreeNodeID
}

// Tree is a variant of the Fyne GUI toolkit,
// which allows creating and working with the tree's data in a node representation.
//
// Tree also provides variants of classic Tree method that allows working with nodes directly.
//
// # Example
//
/*
	package main

	import (
		"fyne.io/fyne/v2"
		"fyne.io/fyne/v2/app"
		"fyne.io/fyne/v2/widget"

		iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	)

	type node struct {
		value string
	}

	func (n node) UID() widget.TreeNodeID {
		return n.value
	}

	func main() {
		a := app.New()
		w := a.NewWindow("Tree Example")
		tree := iwidget.NewTree(
			func(_ bool) fyne.CanvasObject {
				return widget.NewLabel("Template")
			},
			func(n node, _ bool, co fyne.CanvasObject) {
				co.(*widget.Label).SetText(n.value)
			},
		)
		var nodes iwidget.TreeNodes[node]
		root := nodes.MustAdd(iwidget.RootUID, node{"Root"})
		nodes.Add(root, node{"Alpha"})
		nodes.Add(root, node{"Bravo"})
		tree.Set(nodes)
		tree.OpenAllBranches()
		w.SetContent(tree)
		w.Resize(fyne.NewSize(600, 400))
		w.ShowAndRun()
	}
*/
type Tree[T TreeNode] struct {
	widget.Tree

	OnSelectedNode func(n T)

	nodes TreeNodes[T]
}

// NewTree returns a new [Tree] object.
func NewTree[T TreeNode](
	create func(isBranch bool) fyne.CanvasObject,
	update func(n T, isBranch bool, co fyne.CanvasObject),
) *Tree[T] {
	w := &Tree[T]{
		nodes: TreeNodes[T]{},
	}
	w.ChildUIDs = func(uid widget.TreeNodeID) []widget.TreeNodeID {
		return w.nodes.ChildUIDs(uid)
	}
	w.IsBranch = func(uid widget.TreeNodeID) bool {
		return w.nodes.IsBranch(uid)
	}
	w.CreateNode = create
	w.UpdateNode = func(uid widget.TreeNodeID, isBranch bool, co fyne.CanvasObject) {
		n, ok := w.nodes.Node(uid)
		if !ok {
			return
		}
		update(n, isBranch, co)
	}
	w.OnSelected = func(uid widget.TreeNodeID) {
		if w.OnSelectedNode != nil {
			n, ok := w.nodes.Node(uid)
			if !ok {
				return
			}
			w.OnSelectedNode(n)
		}
	}
	w.ExtendBaseWidget(w)
	return w
}

// Clear removes all nodes from the tree.
func (w *Tree[T]) Clear() {
	w.nodes.Clear()
	w.Refresh()
}

// Set updates all nodes of a tree.
func (w *Tree[T]) Set(nodes TreeNodes[T]) {
	w.nodes = nodes
	w.Refresh()
}

// Nodes returns the nodes for a tree.
func (w *Tree[T]) Nodes() TreeNodes[T] {
	return w.nodes
}

// TreeNodes holds all the nodes for rendering a [Tree] widget.
//
// It is designed to make it easier to construct tree data structures for a widget
// by hiding some of the complexity behind a simpler API and by performing sanity checks.
//
// Trees are constructed by adding nodes, which can contain any data
// as long as it complies with the [TreeNode] interface.
//
// Please note that it is not possible to model empty branches
// as nodes without children are interpreted as leafs.
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
		return RootUID, ErrUndefined
	}
	if t.ids == nil || t.parents == nil || t.nodes == nil {
		t.init()
	}
	if parentUID != RootUID {
		_, found := t.nodes[parentUID]
		if !found {
			return RootUID, fmt.Errorf("parent node does not exist: %s", parentUID)
		}
	}
	uid := node.UID()
	if uid == RootUID {
		return RootUID, fmt.Errorf("UID() must not return zero value: %+v", node)
	}
	_, found := t.nodes[uid]
	if found {
		return RootUID, fmt.Errorf("this node already exists: %+v", node)
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

// All returns an iterator over all nodes.
// The nodes will be have no specific order.
func (t TreeNodes[T]) All() iter.Seq[T] {
	return maps.Values(t.nodes)
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

// Equal reports whether two set of nodes are equal.
// Individual nodes are equal when their UIDs are equal.
func (t TreeNodes[T]) Equal(n TreeNodes[T]) bool {
	return reflect.DeepEqual(t.ids, n.ids)
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
		if uid == RootUID {
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
