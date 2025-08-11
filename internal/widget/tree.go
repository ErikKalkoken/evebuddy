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

	data TreeData[T]
}

// NewTree returns a new [Tree] object.
func NewTree[T TreeNode](
	create func(isBranch bool) fyne.CanvasObject,
	update func(n T, isBranch bool, co fyne.CanvasObject),
) *Tree[T] {
	w := &Tree[T]{
		data: TreeData[T]{},
	}
	w.ChildUIDs = func(uid widget.TreeNodeID) []widget.TreeNodeID {
		return w.data.ChildUIDs(uid)
	}
	w.IsBranch = func(uid widget.TreeNodeID) bool {
		return w.data.IsBranch(uid)
	}
	w.CreateNode = create
	w.UpdateNode = func(uid widget.TreeNodeID, isBranch bool, co fyne.CanvasObject) {
		n, ok := w.data.Node(uid)
		if !ok {
			return
		}
		update(n, isBranch, co)
	}
	w.OnSelected = func(uid widget.TreeNodeID) {
		if w.OnSelectedNode != nil {
			n, ok := w.data.Node(uid)
			if !ok {
				return
			}
			w.OnSelectedNode(n)
		}
	}
	w.ExtendBaseWidget(w)
	return w
}

// Clear resets the tree to an empty tree.
func (w *Tree[T]) Clear() {
	w.data.Clear()
	w.Refresh()
}

// Data returns the tree's data.
func (w *Tree[T]) Data() TreeData[T] {
	return w.data
}

// Set replaces the tree's data.
func (w *Tree[T]) Set(data TreeData[T]) {
	w.data = data
	w.Refresh()
}

// TreeData holds the data for rendering a [Tree] widget.
//
// It is designed to make it easier to construct a tree widget
// by providing a tree like API and sanity checks.
//
// Trees are constructed by adding nodes to a root node, which has the UID [TreeRootID].
// The root node always exists and is empty.
// Nodes can contain any data as long as they comply with the [TreeNode] interface.
// The order of added nodes is preserved.
//
// Note that it is not possible to model an empty branch
// as a node without children is interpreted as a leaf.
//
// The zero value is an empty tree ready to use.
type TreeData[T TreeNode] struct {
	children map[widget.TreeNodeID][]widget.TreeNodeID
	nodes    map[widget.TreeNodeID]T
	parents  map[widget.TreeNodeID]widget.TreeNodeID
}

// Add adds a node and return's the UID of the added node.
// It return an error if the node can not be added.
//
// Use [TreeRootID] as parentUID for adding nodes at the top level.
func (t *TreeData[T]) Add(parentUID widget.TreeNodeID, node T) (widget.TreeNodeID, error) {
	if t == nil {
		return TreeRootID, ErrInvalid
	}
	if t.children == nil || t.parents == nil || t.nodes == nil {
		t.init() // init zero value
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

func (t *TreeData[T]) init() {
	t.children = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	t.parents = make(map[widget.TreeNodeID]widget.TreeNodeID)
	t.nodes = make(map[widget.TreeNodeID]T)
}

// All returns an iterator over all nodes.
// The order in which nodes are returned is undefined.
func (t TreeData[T]) All() iter.Seq[T] {
	return maps.Values(t.nodes)
}

// Children returns the direct children of a node.
// It returns an error if the node does not exist.
// The root node always exists and has no children if the tree is empty.
func (t TreeData[T]) Children(uid widget.TreeNodeID) ([]T, error) {
	var nodes []T
	if t.IsEmpty() && uid == TreeRootID {
		return nodes, nil
	}
	if uid != TreeRootID {
		_, found := t.nodes[uid]
		if !found {
			return nil, fmt.Errorf("children for uid: %s: %w", uid, ErrNotFound)
		}
	}
	_, found := t.children[uid]
	if !found {
		return nodes, nil
	}
	for _, id := range t.children[uid] {
		nodes = append(nodes, t.nodes[id])
	}
	return nodes, nil
}

// ChildrenCount returns the number of direct children of a node.
// It returns an error if the node does not exist.
func (t TreeData[T]) ChildrenCount(uid widget.TreeNodeID) (int, error) {
	if t.IsEmpty() && uid == TreeRootID {
		return 0, nil
	}
	_, found := t.children[uid]
	if !found {
		return 0, fmt.Errorf("children count for uid: %s: %w", uid, ErrNotFound)
	}
	return len(t.children[uid]), nil
}

// Clear removes all nodes.
func (t *TreeData[T]) Clear() {
	if t == nil {
		fyne.LogError("Trying to clear a tree with a nil pointer", ErrInvalid)
		return
	}
	if t.IsEmpty() {
		return
	}
	t.init()
}

// Clone returns a clone of itself.
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

// Delete deletes a subtree given by the UID of it's root node.
// It will return [ErrNotFound] if the node does not exist.
// The root node can not be removed.
func (t TreeData[T]) Delete(uid widget.TreeNodeID) error {
	if uid == TreeRootID {
		return fmt.Errorf("can not remove root node: %w", ErrInvalid)
	}
	if t.IsEmpty() {
		return fmt.Errorf("uid: %s: %w", uid, ErrNotFound)
	}
	_, found := t.nodes[uid]
	if !found {
		return fmt.Errorf("uid: %s: %w", uid, ErrNotFound)
	}
	t.delete(uid)
	return nil
}

func (t TreeData[T]) delete(uid widget.TreeNodeID) {
	s, found := t.children[uid]
	if found {
		s2 := slices.Clone(s)
		for _, n := range s2 {
			if n == TreeRootID {
				panic("root ID found in children: " + uid)
			}
			t.delete(n)
		}
		delete(t.children, uid)
	}
	parent, found := t.parents[uid]
	if !found {
		panic("Parent not found for UID: " + uid)
	}
	t.children[parent] = slices.DeleteFunc(t.children[parent], func(x widget.TreeNodeID) bool {
		return x == uid
	})
	delete(t.parents, uid)
	delete(t.nodes, uid)
}

// Equal reports whether two trees are equal. Empty trees are always equal.
func (t TreeData[T]) Equal(td TreeData[T]) bool {
	if t.IsEmpty() && td.IsEmpty() {
		return true
	}
	return reflect.DeepEqual(t.children, td.children) &&
		reflect.DeepEqual(t.parents, td.parents) &&
		reflect.DeepEqual(t.nodes, td.nodes)
}

// IsBranch reports whether a node is a branch.
func (t TreeData[T]) IsBranch(uid widget.TreeNodeID) bool {
	_, found := t.children[uid]
	return found
}

// IsEmpty reports whether the tree has any nodes other then the root node.
func (t TreeData[T]) IsEmpty() bool {
	return len(t.nodes) == 0
}

// MustAdd tries to add a node and panics if the node can not be added.
func (t *TreeData[T]) MustAdd(parentUID widget.TreeNodeID, node T) widget.TreeNodeID {
	uid, err := t.Add(parentUID, node)
	if err != nil {
		panic(err)
	}
	return uid
}

// Node returns a node by UID and reports whether it exists.
// The root node always exists and is empty.
//
// Note that when using this method with a Fyne widget it is possible,
// that a UID forwarded by the widget no longer exists due to race conditions.
// It is therefore recommended to always check the ok value.
func (t TreeData[T]) Node(uid widget.TreeNodeID) (node T, ok bool) {
	if uid == TreeRootID {
		var zero T
		return zero, true
	}
	node, ok = t.nodes[uid]
	return
}

// Parent returns the UID of the parent node and reports if it exists.
func (t TreeData[T]) Parent(uid widget.TreeNodeID) (parent widget.TreeNodeID, ok bool) {
	parent, ok = t.parents[uid]
	return
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
	for _, id := range t.ChildUIDs(uid) {
		t.print(id, indent, len(t.ChildUIDs(id)) == 0)
	}
}

// Size returns the total number of nodes in the tree including the root node.
func (t TreeData[T]) Size() int {
	return len(t.nodes) + 1
}

// String implements the stringer interface.
func (t TreeData[T]) String() string {
	return fmt.Sprintf("{nodes %+v, children: %+v}", t.nodes, t.children)
}
