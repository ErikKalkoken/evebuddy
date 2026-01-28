package widget

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Tree2 is a thin wrapper around Fyne's Tree widget that allows building
// and maintaining trees with generic nodes.
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
		text string
	}

	func main() {
		a := app.New()
		w := a.NewWindow("Tree2 Example")
		tree := iwidget.NewTree2(
			func(_ bool) fyne.CanvasObject {
				return widget.NewLabel("Template")
			},
			func(n *node, _ bool, co fyne.CanvasObject) {
				co.(*widget.Label).SetText(n.text)
			},
		)
		var td iwidget.TreeData2[node]
		top := &node{"Top"}
		td.Add(nil, top)
		nodes.Add(root, &node{"Alpha"})
		nodes.Add(root, &node{"Bravo"})
		tree.Set(nodes)
		w.SetContent(tree)
		w.Resize(fyne.NewSize(600, 400))
		w.ShowAndRun()
	}
*/
type Tree2[T any] struct {
	widget.Tree

	OnSelectedNode func(n *T)

	td TreeData2[T]
}

// NewTree2 returns a new [Tree2] object.
func NewTree2[T any](
	create func(isBranch bool) fyne.CanvasObject,
	update func(n *T, isBranch bool, co fyne.CanvasObject),
) *Tree2[T] {
	w := &Tree2[T]{
		td: newTreeData[T](),
	}
	w.ChildUIDs = func(uid widget.TreeNodeID) []widget.TreeNodeID {
		return w.td.children[uid]
	}
	w.IsBranch = func(uid widget.TreeNodeID) bool {
		if uid == TreeRootID {
			return true
		}
		return w.td.isBranchNode[uid]
	}
	w.CreateNode = create
	w.UpdateNode = func(uid widget.TreeNodeID, isBranch bool, co fyne.CanvasObject) {
		n, ok := w.td.Node(uid)
		if !ok {
			return
		}
		update(n, isBranch, co)
	}
	w.OnSelected = func(uid widget.TreeNodeID) {
		if w.OnSelectedNode != nil {
			n, ok := w.td.Node(uid)
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
func (w *Tree2[T]) Clear() {
	w.td.Clear()
	w.Refresh()
}

// Data returns the tree's data.
func (w *Tree2[T]) Data() TreeData2[T] {
	return w.td
}

// Set replaces the tree's data.
func (w *Tree2[T]) Set(data TreeData2[T]) {
	w.td = data
	w.Refresh()
}

func (w *Tree2[T]) OpenBranchNode(n *T) {
	w.callWhenFound(n, w.OpenBranch)
}

func (w *Tree2[T]) CloseBranchNode(n *T) {
	w.callWhenFound(n, w.CloseBranch)
}

// SelectNode selects node n.
// An invalid node will be ignored. The root node can not be selected.
func (w *Tree2[T]) SelectNode(n *T) {
	w.callWhenFound(n, w.Select)
}

func (w *Tree2[T]) ScrollToNode(n *T) {
	w.callWhenFound(n, w.ScrollTo)
}

func (w *Tree2[T]) ToggleBranchNode(n *T) {
	w.callWhenFound(n, w.ToggleBranch)
}

func (w *Tree2[T]) callWhenFound(n *T, f func(widget.TreeNodeID)) {
	uid, ok := w.td.uidLookup[n]
	if ok {
		f(uid)
	}
}

// TreeData2 holds the data for rendering a [Tree2] widget.
//
// It is designed to make it easier to construct a tree widget
// by providing a graph like API and sanity checks.
//
// Trees are constructed by adding nodes to a virtual root node.
// Nodes can be any struct.
// The virtual root node always exists and is represented with nil.
//
// The zero value is an empty tree ready to use.
type TreeData2[T any] struct {
	children     map[widget.TreeNodeID][]widget.TreeNodeID
	id           int
	isBranchNode map[widget.TreeNodeID]bool
	nodes        map[widget.TreeNodeID]*T
	parents      map[widget.TreeNodeID]widget.TreeNodeID
	uidLookup    map[*T]widget.TreeNodeID
}

func newTreeData[T any]() TreeData2[T] {
	td := TreeData2[T]{}
	td.init()
	return td
}

func (t *TreeData2[T]) init() {
	t.children = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	t.id = 0
	t.isBranchNode = make(map[widget.TreeNodeID]bool)
	t.nodes = make(map[widget.TreeNodeID]*T)
	t.parents = make(map[widget.TreeNodeID]widget.TreeNodeID)
	t.uidLookup = make(map[*T]widget.TreeNodeID)
}

// Add adds a node to the tree.
// The order in which nodes are added is preserved.
// It returns an error when the node can not be added.
//
// The root node is represented as nil parent.
func (t *TreeData2[T]) Add(parent *T, node *T, isBranch bool) error {
	if t == nil || node == nil {
		return ErrInvalid
	}
	if t.children == nil {
		t.init() // init zero value
	}
	parentUID, ok := t.UID(parent)
	if !ok {
		return fmt.Errorf("parent not found: %w", ErrNotFound)
	}
	t.id++
	uid := strconv.Itoa(t.id)
	t.children[parentUID] = append(t.children[parentUID], uid)
	t.nodes[uid] = node
	t.parents[uid] = parentUID
	t.isBranchNode[uid] = isBranch
	t.uidLookup[node] = uid
	return nil
}

// All returns an iterator over all nodes.
// The order in which nodes are returned is undefined.
func (t TreeData2[T]) All() iter.Seq[*T] {
	return maps.Values(t.nodes)
}

// Children returns a new slice with the direct children of a node
// and reports whether the node exists.
// The children are returns in the same order as they were added.
// When a node does not exit it returns an empty slice.
func (t TreeData2[T]) Children(node *T) []*T {
	nodes := make([]*T, 0)
	uid, ok := t.UID(node)
	if !ok {
		return nodes
	}
	for _, id := range t.children[uid] {
		nodes = append(nodes, t.nodes[id])
	}
	return nodes
}

// ChildrenCount returns the number of direct children of a node
// and reports whether the node exists.
func (t TreeData2[T]) ChildrenCount(node *T) (int, bool) {
	uid, found := t.UID(node)
	if !found {
		return 0, false
	}
	return len(t.children[uid]), true
}

// Clear removes all nodes.
func (t *TreeData2[T]) Clear() {
	if t == nil {
		fyne.LogError("Trying to clear a nil tree", ErrInvalid)
		return
	}
	if t.IsEmpty() {
		return
	}
	t.init()
}

// Delete deletes a subtree given by the UID of it's root node.
// It will return [ErrNotFound] if the node does not exist.
// The root node can not be removed.
func (t TreeData2[T]) Delete(node *T) error {
	if node == nil {
		return fmt.Errorf("Delete: can not remove root node: %w", ErrInvalid)
	}
	if t.IsEmpty() {
		return fmt.Errorf("Delete: %w", ErrNotFound)
	}
	uid, found := t.uidLookup[node]
	if !found {
		return fmt.Errorf("uid: %s: %w", uid, ErrNotFound)
	}
	t.delete(uid)
	return nil
}

func (t TreeData2[T]) delete(uid widget.TreeNodeID) {
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
	delete(t.isBranchNode, uid)
	n, found := t.nodes[uid]
	if found {
		delete(t.uidLookup, n)
	}
}

// IsBranch reports whether a node is a branch and reports whether the node exists.
func (t TreeData2[T]) IsBranch(node *T) (isBranch bool, ok bool) {
	uid, ok := t.UID(node)
	if !ok {
		return false, false
	}
	return t.isBranchNode[uid], true
}

// SetBranch sets the branch state for a node and reports whether it exists.
// The root node can not be changed.
func (t *TreeData2[T]) SetBranch(node *T, isBranch bool) bool {
	if t == nil {
		fyne.LogError("Trying to set a branch in a nil tree", ErrInvalid)
		return false
	}
	uid, ok := t.uidLookup[node]
	if !ok {
		return false
	}
	t.isBranchNode[uid] = isBranch
	return true
}

// IsEmpty reports whether the tree has any nodes (other then the root node).
func (t TreeData2[T]) IsEmpty() bool {
	return len(t.nodes) == 0
}

// Node returns a node by UID and reports whether it exists.
// The root node will be returned as nil.
func (t TreeData2[T]) Node(uid widget.TreeNodeID) (node *T, ok bool) {
	if uid == TreeRootID {
		return nil, true
	}
	node, ok = t.nodes[uid]
	return
}

// UID returns the UID for a node and reports whether the operation succeeded.
// Nil represents the root node and is valid.
func (t TreeData2[T]) UID(node *T) (uid widget.TreeNodeID, ok bool) {
	if node == nil {
		return TreeRootID, true
	}
	uid, ok = t.uidLookup[node]
	return
}

// MustUID is similar to UID(), but will panic if the node does not exist.
func (t TreeData2[T]) MustUID(node *T) widget.TreeNodeID {
	uid, ok := t.UID(node)
	if !ok {
		panic("UID not found")
	}
	return uid
}

// Exists reports whether a node exists.
// Nil will also return represents the root node and will also return true.
func (t TreeData2[T]) Exists(node *T) bool {
	if node == nil {
		return true
	}
	_, ok := t.uidLookup[node]
	return ok
}

// Parent returns the parent of a node and reports whether the operation succeeded.
func (t TreeData2[T]) Parent(node *T) (*T, bool) {
	uid, ok := t.uidLookup[node]
	if !ok {
		return nil, false
	}
	parent := t.parents[uid]
	return t.nodes[parent], true
}

// Path returns the nodes between a given node and the root.
func (t TreeData2[T]) Path(node *T) []*T {
	path := make([]*T, 0)
	uid, ok := t.uidLookup[node]
	if !ok {
		return path
	}
	for {
		uid = t.parents[uid]
		if uid == TreeRootID {
			break
		}
		path = append(path, t.nodes[uid])
	}
	slices.Reverse(path)
	return path
}

// Print prints a tree to the console.
// Nodes should comply with the stringer interface for best results.
func (t TreeData2[T]) Print(node *T) {
	if uid, ok := t.UID(node); ok {
		t.print(uid, "", false)
		fmt.Println()
	}
}

func (t TreeData2[T]) print(uid widget.TreeNodeID, indent string, last bool) {
	var s string
	if uid == TreeRootID {
		s = "ROOT"
	} else {
		n, _ := t.Node(uid)
		s = fmt.Sprint(n)
	}
	fmt.Println(indent + "+- " + s)
	if last {
		indent += "   "
	} else {
		indent += "|  "
	}
	for _, id := range t.children[uid] {
		t.print(id, indent, len(t.children[id]) == 0)
	}
}

// Size returns the total number of nodes in the tree excluding the root node.
func (t TreeData2[T]) Size() int {
	return len(t.nodes)
}

// String implements the stringer interface.
func (t TreeData2[T]) String() string {
	return fmt.Sprintf("{nodes %+v, children: %+v}", t.nodes, t.children)
}
