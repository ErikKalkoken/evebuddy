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
	TreeRootID widget.TreeNodeID = "" // UID of the root node in a Tree widget
)

var (
	ErrInvalid  = errors.New("invalid operation")
	ErrNotFound = errors.New("not found")
)

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

	nodes TreeData[T]
}

// NewTree returns a new [Tree] object.
func NewTree[T TreeNode](
	create func(isBranch bool) fyne.CanvasObject,
	update func(n T, isBranch bool, co fyne.CanvasObject),
) *Tree[T] {
	w := &Tree[T]{
		nodes: TreeData[T]{},
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
func (w *Tree[T]) Set(nodes TreeData[T]) {
	w.nodes = nodes
	w.Refresh()
}

// Nodes returns the nodes for a tree.
func (w *Tree[T]) Nodes() TreeData[T] {
	return w.nodes
}

// TreeData holds all the nodes for rendering a [Tree] widget.
//
// It is designed to make it easier to construct tree data structures for a widget
// by hiding some of the complexity behind a simpler API and by performing sanity checks.
//
// Trees are constructed by adding nodes to a root, which is identified by [TreeRootID]
// Nodes can contain any data as long as it complies with the [TreeNode] interface.
//
// Note that it is not possible to model empty branches
// as nodes without children are interpreted as leafs.
//
// The zero value is an empty tree ready to use.
// This type is not thread safe.
type TreeData[T TreeNode] struct {
	children map[widget.TreeNodeID][]widget.TreeNodeID
	nodes    map[widget.TreeNodeID]T
	parents  map[widget.TreeNodeID]widget.TreeNodeID
}

// Add adds a node safely. It returns it's UID or an error if the node can not be added.
//
// Use [TreeRootID] as parentUID for adding nodes at the top level.
// Nodes will be rendered in the same order as they are added.
func (t *TreeData[T]) Add(parentUID widget.TreeNodeID, node T) (widget.TreeNodeID, error) {
	if t == nil {
		return TreeRootID, ErrInvalid
	}
	if t.isZero() {
		t.init()
	}
	if parentUID != TreeRootID {
		_, found := t.nodes[parentUID]
		if !found {
			return TreeRootID, fmt.Errorf("parent node does not exist: %s: %w", parentUID, ErrInvalid)
		}
	}
	uid := node.UID()
	if uid == TreeRootID {
		return TreeRootID, fmt.Errorf("UID() must not return zero value: %+v: %w", node, ErrInvalid)
	}
	_, found := t.nodes[uid]
	if found {
		return TreeRootID, fmt.Errorf("this node already exists: %+v: %w", node, ErrInvalid)
	}
	t.children[parentUID] = append(t.children[parentUID], uid)
	t.nodes[uid] = node
	t.parents[uid] = parentUID
	return uid, nil
}

func (t TreeData[T]) isZero() bool {
	return t.children == nil || t.parents == nil || t.nodes == nil
}

func (t *TreeData[T]) init() {
	t.children = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	t.parents = make(map[widget.TreeNodeID]widget.TreeNodeID)
	t.nodes = make(map[widget.TreeNodeID]T)
}

// All returns an iterator over all nodes in no specific order.
func (t TreeData[T]) All() iter.Seq[T] {
	return maps.Values(t.nodes)
}

// Clear removes all nodes.
func (t *TreeData[T]) Clear() {
	if t == nil {
		panic(ErrInvalid)
	}
	t.init()
}

// Clone returns a clone of a TreeData object.
func (t TreeData[T]) Clone() TreeData[T] {
	t2 := TreeData[T]{
		parents:  maps.Clone(t.parents),
		nodes:    maps.Clone(t.nodes),
		children: make(map[widget.TreeNodeID][]widget.TreeNodeID),
	}
	for k, v := range t.children {
		t2.children[k] = slices.Clone(v)
	}
	return t2
}

// ChildUIDs returns the UIDs of the direct children of a node.
// It returns an empty slice if the node has no children or the node does not exist.
func (t TreeData[T]) ChildUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	return t.children[uid]
}

// Remove removes a subtree given by uid as it's root.
func (t TreeData[T]) Remove(uid widget.TreeNodeID) error {
	if t.Size() == 0 {
		return fmt.Errorf("uid: %s: %w", uid, ErrNotFound)
	}
	_, found := t.nodes[uid]
	if !found {
		return fmt.Errorf("uid: %s: %w", uid, ErrNotFound)
	}
	return t.removeNode(uid)
}

func (t TreeData[T]) removeNode(uid widget.TreeNodeID) error {
	s, found := t.children[uid]
	if found {
		s2 := slices.Clone(s)
		for _, n := range s2 {
			if n == "" {
				return fmt.Errorf("root ID found in children: %s: %w", uid, ErrInvalid)
			}
			err := t.removeNode(n)
			if err != nil {
				return err
			}
		}
		delete(t.children, uid)
	}
	parent, found := t.parents[uid]
	if !found {
		return fmt.Errorf("parent not found: %s: %w", uid, ErrInvalid)
	}
	t.children[parent] = slices.DeleteFunc(t.children[parent], func(x widget.TreeNodeID) bool {
		return x == uid
	})
	delete(t.parents, uid)
	delete(t.nodes, uid)
	return nil
}

// Equal reports whether two trees are equal.
func (t TreeData[T]) Equal(n TreeData[T]) bool {
	if t.Size() == 0 && n.Size() == 0 {
		return true
	}
	return reflect.DeepEqual(t.children, n.children) &&
		reflect.DeepEqual(t.parents, n.parents) &&
		reflect.DeepEqual(t.nodes, n.nodes)
}

// IsBranch reports whether a node is a branch.
func (t TreeData[T]) IsBranch(uid widget.TreeNodeID) bool {
	_, found := t.children[uid]
	return found
}

// MustAdd adds a node or panics if the node can not be added.
func (t *TreeData[T]) MustAdd(parentUID widget.TreeNodeID, node T) widget.TreeNodeID {
	uid, err := t.Add(parentUID, node)
	if err != nil {
		panic(err)
	}
	return uid
}

// MustNode returns a node or panics if the node does not exist.
// This method mainly exists to simplify test code and should not be used in production code.
func (t TreeData[T]) MustNode(uid widget.TreeNodeID) T {
	v, ok := t.Node(uid)
	if !ok {
		panic(fmt.Sprintf("node %s does not exist", uid))
	}
	return v
}

// Node returns a node by UID and reports whether it exists.
// The root node can not be accessed.
//
// Note that when using this method with a Fyne widget it is possible,
// that a UID forwarded by the widget no longer exists due to race conditions.
// It is therefore recommended to always check the ok value.
func (t TreeData[T]) Node(uid widget.TreeNodeID) (node T, ok bool) {
	node, ok = t.node(uid)
	return
}

// node returns a node by UID and reports whether it exists.
func (t TreeData[T]) node(uid widget.TreeNodeID) (T, bool) {
	v, ok := t.nodes[uid]
	return v, ok
}

// Path returns the UIDs of nodes between a given node and the root.
func (t TreeData[T]) Path(uid widget.TreeNodeID) []widget.TreeNodeID {
	path := make([]widget.TreeNodeID, 0)
	for {
		uid = t.parents[uid]
		if uid == TreeRootID {
			break
		}
		path = append(path, uid)
	}
	slices.Reverse(path)
	return path
}

// Parent returns the UID of the parent node.
func (t TreeData[T]) Parent(uid widget.TreeNodeID) (parent widget.TreeNodeID, ok bool) {
	parent, ok = t.parents[uid]
	return
}

// Size returns the total number of nodes in the tree, excluding the root node.
func (t TreeData[T]) Size() int {
	return len(t.nodes)
}

// RootChildrenCount returns the number of children of the root node.
func (t TreeData[T]) RootChildrenCount() int {
	_, found := t.children[TreeRootID]
	if !found {
		return 0
	}
	return len(t.children[TreeRootID])
}

func (t TreeData[T]) Children(uid widget.TreeNodeID) ([]T, error) {
	var n []T
	_, found := t.children[uid]
	if !found {
		return nil, fmt.Errorf("uid: %s: %w", uid, ErrNotFound)
	}
	for _, cUID := range t.children[uid] {
		n = append(n, t.nodes[cUID])
	}
	return n, nil
}

// Dump prints a dump of the content. This is intended for debugging.
func (t TreeData[T]) Dump() {
	fmt.Printf("nodes: %+v\n", t.nodes)
	fmt.Printf("parents: %+v\n", t.parents)
	fmt.Printf("children: %+v\n", t.children)
	fmt.Println()
}

// Print prints a tree to the console.
func (t TreeData[T]) Print(uid widget.TreeNodeID) {
	t.print(uid, "", false)
	fmt.Println()
}

func (t TreeData[T]) print(uid widget.TreeNodeID, indent string, last bool) {
	var s string
	if uid == TreeRootID {
		s = "ROOT"
	} else {
		s = uid
	}
	fmt.Println(indent + "+- " + s)
	if last {
		indent += "   "
	} else {
		indent += "|  "
	}
	for _, cUID := range t.ChildUIDs(uid) {
		t.print(cUID, indent, len(t.ChildUIDs(cUID)) == 0)
	}
}
