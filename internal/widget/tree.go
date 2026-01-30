package widget

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"

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

// Tree is an extension of Fyne's Tree widget that allows to create trees
// with generic nodes.
//
// It also provides wrappers for most tree methods that work directly
// with nodes instead of tree IDs.
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
		w := a.NewWindow("Tree Example")
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
type Tree[T any] struct {
	widget.Tree

	OnSelectedNode func(n *T)

	td TreeData[T]
}

// NewTree returns a new Tree2 object.
func NewTree[T any](
	create func(isBranch bool) fyne.CanvasObject,
	update func(n *T, isBranch bool, co fyne.CanvasObject),
) *Tree[T] {
	w := &Tree[T]{
		td: newTreeData[T](),
	}
	w.ChildUIDs = func(uid widget.TreeNodeID) []widget.TreeNodeID {
		return w.td.children[uid]
	}
	w.IsBranch = func(uid widget.TreeNodeID) bool {
		if uid == TreeRootID {
			return true
		}
		return len(w.td.children[uid]) > 0
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

// Clear removes all nodes of the tree.
func (w *Tree[T]) Clear() {
	w.td.Clear()
	w.Refresh()
}

// Data returns the tree's data.
func (w *Tree[T]) Data() TreeData[T] {
	return w.td
}

// Set replaces the tree's data.
func (w *Tree[T]) Set(data TreeData[T]) {
	w.td = data
	w.Refresh()
}

// OpenBranchNode opens branch node n.
func (w *Tree[T]) OpenBranchNode(n *T) {
	w.callWhenFound(n, w.OpenBranch)
}

// CloseBranchNode closes branch node n.
func (w *Tree[T]) CloseBranchNode(n *T) {
	w.callWhenFound(n, w.CloseBranch)
}

// SelectNode marks the node n to be selected.
func (w *Tree[T]) SelectNode(n *T) {
	w.callWhenFound(n, w.Select)
}

// ScrollToNode scrolls to node n.
func (w *Tree[T]) ScrollToNode(n *T) {
	w.callWhenFound(n, w.ScrollTo)
}

// ToggleBranchNode flips the state of branch node n.
func (w *Tree[T]) ToggleBranchNode(n *T) {
	w.callWhenFound(n, w.ToggleBranch)
}

func (w *Tree[T]) callWhenFound(n *T, f func(widget.TreeNodeID)) {
	uid, ok := w.td.uidLookup[n]
	if ok {
		f(uid)
	}
}

// TreeData represents the tree structure for rendering a [Tree] widget.
//
// The tree is constructed by adding nodes to a virtual root node.
// The root node always exists and represented by a nil node.
// Nodes are stores as pointers and can be changed after the have been added.
//
// The zero value is an empty tree ready to use.
type TreeData[T any] struct {
	children  map[widget.TreeNodeID][]widget.TreeNodeID
	id        int
	nodes     map[widget.TreeNodeID]*T
	parents   map[widget.TreeNodeID]widget.TreeNodeID
	uidLookup map[*T]widget.TreeNodeID
}

func newTreeData[T any]() TreeData[T] {
	td := TreeData[T]{}
	td.init()
	return td
}

func (td *TreeData[T]) init() {
	td.children = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	td.id = 0
	td.nodes = make(map[widget.TreeNodeID]*T)
	td.parents = make(map[widget.TreeNodeID]widget.TreeNodeID)
	td.uidLookup = make(map[*T]widget.TreeNodeID)
}

// Add adds a node to the tree.
// The order in which nodes are added is preserved.
// Add performs sanity checks to ensure the resulting tree structure is valid
// and returns an error when a problem was found.
//
// A nil parent represents the root node.
func (td *TreeData[T]) Add(parent *T, node *T) error {
	if td == nil || node == nil {
		return ErrInvalid
	}
	if td.children == nil {
		td.init() // init zero value
	}
	parentUID, ok := td.UID(parent)
	if !ok {
		return fmt.Errorf("parent not found: %w", ErrNotFound)
	}
	td.id++
	uid := strconv.Itoa(td.id)
	td.children[parentUID] = append(td.children[parentUID], uid)
	td.nodes[uid] = node
	td.parents[uid] = parentUID
	td.uidLookup[node] = uid
	return nil
}

// Children returns a new slice with the direct children of a node
// and reports whether the node exists.
// The children are returns in the same order as they were added.
// When the node was not found a nil slice is returned.
func (td TreeData[T]) Children(parent *T) []*T {
	uid, ok := td.UID(parent)
	if !ok {
		return nil
	}
	nodes := make([]*T, 0)
	for _, id := range td.children[uid] {
		nodes = append(nodes, td.nodes[id])
	}
	return nodes
}

// ChildrenCount returns the number of direct children of a node
// and reports whether the node was found.
func (td TreeData[T]) ChildrenCount(node *T) (int, bool) {
	uid, found := td.UID(node)
	if !found {
		return 0, false
	}
	return len(td.children[uid]), true
}

// Clear removes all nodes.
func (td *TreeData[T]) Clear() {
	if td == nil {
		fyne.LogError("Trying to clear a nil tree", ErrInvalid)
		return
	}
	if td.IsEmpty() {
		return
	}
	td.init()
}

// Clone returns a shallow copy of the tree data which uses the same node objects.
// The clone can be used to modify the tree structure
// and then update the tree widget with the result in one operation.
func (td TreeData[T]) Clone() TreeData[T] {
	t2 := TreeData[T]{
		children:  make(map[widget.TreeNodeID][]widget.TreeNodeID),
		id:        td.id,
		nodes:     maps.Clone(td.nodes),
		parents:   maps.Clone(td.parents),
		uidLookup: maps.Clone(td.uidLookup),
	}
	for k, v := range td.children {
		t2.children[k] = slices.Clone(v)
	}
	return t2
}

// Delete deletes a subtree given by the UID of it's root node
// It will return an error if the node can not be deleted.
// The root node can not be removed.
func (td TreeData[T]) Delete(node *T) error {
	if node == nil {
		return fmt.Errorf("Delete: can not remove root node: %w", ErrInvalid)
	}
	uid, found := td.uidLookup[node]
	if !found {
		return fmt.Errorf("Delete: uid %s: %w", uid, ErrNotFound)
	}
	td.delete(uid)
	return nil
}

func (td TreeData[T]) delete(uid widget.TreeNodeID) {
	s, found := td.children[uid]
	if found {
		s2 := slices.Clone(s)
		for _, n := range s2 {
			if n == TreeRootID {
				fyne.LogError("root ID found in children: "+uid, ErrInvalid)
				return
			}
			td.delete(n)
		}
		delete(td.children, uid)
	}
	parent, found := td.parents[uid]
	if !found {
		fyne.LogError("Parent not found for UID: "+uid, ErrInvalid)
		return
	}
	td.children[parent] = slices.DeleteFunc(td.children[parent], func(x widget.TreeNodeID) bool {
		return x == uid
	})
	delete(td.parents, uid)
	delete(td.nodes, uid)
	n, found := td.nodes[uid]
	if found {
		delete(td.uidLookup, n)
	}
}

// Exists reports whether a node exists.
// Nil will also return represents the root node and will also return true.
func (td TreeData[T]) Exists(node *T) bool {
	if node == nil {
		return true
	}
	_, ok := td.uidLookup[node]
	return ok
}

// IsEmpty reports whether the tree has any nodes (other then the root node).
func (td TreeData[T]) IsEmpty() bool {
	return len(td.nodes) == 0
}

// Node returns a node by UID and reports whether it was found.
// The root node will be returned as nil.
func (td TreeData[T]) Node(uid widget.TreeNodeID) (node *T, ok bool) {
	if uid == TreeRootID {
		return nil, true
	}
	node, ok = td.nodes[uid]
	return
}

// Parent returns the parent of a node and reports whether the operation succeeded.
func (td TreeData[T]) Parent(node *T) (*T, bool) {
	uid, ok := td.uidLookup[node]
	if !ok {
		return nil, false
	}
	parent := td.parents[uid]
	return td.nodes[parent], true
}

// Path returns the path from parent to n.
// The path includes parent (except root) and n.
// Parent must be an ancestor of n or nil for the root node.
// Returns a nil slice when no path can be found.
func (td TreeData[T]) Path(parent, n *T) []*T {
	if n == nil {
		return nil
	}
	aUID, ok := td.UID(n)
	if !ok {
		return nil
	}
	bUID, ok := td.UID(parent)
	if !ok {
		return nil
	}
	path := make([]*T, 0)
	path = append(path, n)
	for {
		aUID = td.parents[aUID]
		if aUID != TreeRootID {
			path = append(path, td.nodes[aUID])
		}
		if aUID == bUID {
			break
		}
	}
	slices.Reverse(path)
	return path
}

// AllPaths returns a slice of paths from parent to all leafs.
// This is a type of linearization and can be useful for comparing trees in tests.
//
// T is expected to implement the stringer interface.
// Will return all paths from root when parent is nil.
func (td TreeData[T]) AllPaths(parent *T) [][]string {
	all := make([][]string, 0)
	td.Walk(parent, func(n *T) bool {
		if c, ok := td.ChildrenCount(n); ok && c == 0 {
			p := make([]string, 0)
			for _, x := range td.Path(parent, n) {
				p = append(p, fmt.Sprint(x))
			}
			all = append(all, p)
		}
		return true
	})
	return all
}

// Print prints a sub tree of node n to the console.
// T is expected to implement the stringer interface.
func (td TreeData[T]) Print(n *T) {
	if uid, ok := td.UID(n); ok {
		td.print(uid, "", false)
		fmt.Println()
	}
}

func (td TreeData[T]) print(uid widget.TreeNodeID, indent string, last bool) {
	var s string
	if uid == TreeRootID {
		s = "ROOT"
	} else {
		n, _ := td.Node(uid)
		s = fmt.Sprint(n)
	}
	fmt.Println(indent + "+- " + s)
	if last {
		indent += "   "
	} else {
		indent += "|  "
	}
	for _, id := range td.children[uid] {
		td.print(id, indent, len(td.children[id]) == 0)
	}
}

// Size returns the number of nodes in the tree excluding the virtual root node.
func (td TreeData[T]) Size() int {
	return len(td.nodes)
}

// UID returns the UID for a node and reports whether it was found.
// Nil represents the root node and is valid.
func (td TreeData[T]) UID(node *T) (uid widget.TreeNodeID, ok bool) {
	if node == nil {
		return TreeRootID, true
	}
	uid, ok = td.uidLookup[node]
	return
}

// Walk walks the sub tree of parent, calling f for each node in the tree,
// including parent (except if parent is root).
//
// The nodes are walked in depth first search order.
// Walk starts at the root when parent is nil.
// Walk does nothing if parent is not found.
// The caller can return true to continue walking and false to exit early.
func (td TreeData[T]) Walk(parent *T, f func(n *T) bool) {
	var traverse func(widget.TreeNodeID) bool
	traverse = func(curr widget.TreeNodeID) bool {
		if curr != TreeRootID {
			n, ok := td.nodes[curr]
			if !ok {
				return true
			}
			if !f(n) {
				return false
			}
		}
		for _, c := range td.children[curr] {
			if !traverse(c) {
				return false
			}
		}
		return true
	}
	uid, ok := td.UID(parent)
	if !ok {
		return
	}
	traverse(uid)
}
