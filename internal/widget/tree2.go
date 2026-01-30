package widget

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// Tree2 is an extension of Fyne's Tree widget that allows to create trees
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

// NewTree2 returns a new Tree2 object.
func NewTree2[T any](
	create func(isBranch bool) fyne.CanvasObject,
	update func(n *T, isBranch bool, co fyne.CanvasObject),
) *Tree2[T] {
	w := &Tree2[T]{
		td: newTreeData2[T](),
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

// TreeData2 represents the tree structure for rendering a [Tree2] widget.
//
// The tree is constructed by adding nodes to a virtual root node.
// The root node always exists and represented by a nil node.
// Nodes are stores as pointers and can be changed after the have been added.
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

func newTreeData2[T any]() TreeData2[T] {
	td := TreeData2[T]{}
	td.init()
	return td
}

func (td *TreeData2[T]) init() {
	td.children = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	td.id = 0
	td.isBranchNode = make(map[widget.TreeNodeID]bool)
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
func (td *TreeData2[T]) Add(parent *T, node *T, isBranch bool) error {
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
	td.isBranchNode[uid] = isBranch
	td.uidLookup[node] = uid
	return nil
}

// All returns an iterator over all nodes of a subtree.
// The nodes are returned in depth first search order.
// Will do nothing if the node is not found.
// The nil node represents the root.
func (td TreeData2[T]) All(parent *T) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		var traverse func(widget.TreeNodeID) bool
		traverse = func(curr widget.TreeNodeID) bool {
			if curr != TreeRootID {
				n, ok := td.nodes[curr]
				if !ok {
					return true
				}
				if !yield(n) {
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
}

// Children returns a new slice with the direct children of a node
// and reports whether the node exists.
// The children are returns in the same order as they were added.
// When the node was not found a nil slice is returned.
func (td TreeData2[T]) Children(node *T) []*T {
	uid, ok := td.UID(node)
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
func (td TreeData2[T]) ChildrenCount(node *T) (int, bool) {
	uid, found := td.UID(node)
	if !found {
		return 0, false
	}
	return len(td.children[uid]), true
}

// Clear removes all nodes.
func (td *TreeData2[T]) Clear() {
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
func (td TreeData2[T]) Clone() TreeData2[T] {
	t2 := TreeData2[T]{
		children:     make(map[widget.TreeNodeID][]widget.TreeNodeID),
		id:           td.id,
		isBranchNode: maps.Clone(td.isBranchNode),
		nodes:        maps.Clone(td.nodes),
		parents:      maps.Clone(td.parents),
		uidLookup:    maps.Clone(td.uidLookup),
	}
	for k, v := range td.children {
		t2.children[k] = slices.Clone(v)
	}
	return t2
}

// Delete deletes a subtree given by the UID of it's root node
// It will return an error if the node can not be deleted.
// The root node can not be removed.
func (td TreeData2[T]) Delete(node *T) error {
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

func (td TreeData2[T]) delete(uid widget.TreeNodeID) {
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
	delete(td.isBranchNode, uid)
	n, found := td.nodes[uid]
	if found {
		delete(td.uidLookup, n)
	}
}

// IsBranch reports whether a node is a branch and it exists.
func (td TreeData2[T]) IsBranch(node *T) (isBranch bool, ok bool) {
	uid, ok := td.UID(node)
	if !ok {
		return false, false
	}
	return td.isBranchNode[uid], true
}

// SetBranch sets the branch state for a node.
// The root node can not be changed.
// Does nothing if the node is not found.
func (td *TreeData2[T]) SetBranch(node *T, isBranch bool) bool {
	if td == nil {
		fyne.LogError("Trying to set a branch in a nil tree", ErrInvalid)
		return false
	}
	uid, ok := td.uidLookup[node]
	if !ok {
		return false
	}
	td.isBranchNode[uid] = isBranch
	return true
}

// IsEmpty reports whether the tree has any nodes (other then the root node).
func (td TreeData2[T]) IsEmpty() bool {
	return len(td.nodes) == 0
}

// Node returns a node by UID and reports whether it was found.
// The root node will be returned as nil.
func (td TreeData2[T]) Node(uid widget.TreeNodeID) (node *T, ok bool) {
	if uid == TreeRootID {
		return nil, true
	}
	node, ok = td.nodes[uid]
	return
}

// UID returns the UID for a node and reports whether it was found.
// Nil represents the root node and is valid.
func (td TreeData2[T]) UID(node *T) (uid widget.TreeNodeID, ok bool) {
	if node == nil {
		return TreeRootID, true
	}
	uid, ok = td.uidLookup[node]
	return
}

// Exists reports whether a node exists.
// Nil will also return represents the root node and will also return true.
func (td TreeData2[T]) Exists(node *T) bool {
	if node == nil {
		return true
	}
	_, ok := td.uidLookup[node]
	return ok
}

// Parent returns the parent of a node and reports whether the operation succeeded.
func (td TreeData2[T]) Parent(node *T) (*T, bool) {
	uid, ok := td.uidLookup[node]
	if !ok {
		return nil, false
	}
	parent := td.parents[uid]
	return td.nodes[parent], true
}

// Path returns the nodes between node a and the root.
// Returns a nil slice when node was not found.
func (td TreeData2[T]) Path(node *T) []*T {
	return td.path(node, nil)
}

// path returns the nodes between the nodes a and parent,
// where b must be an ancestor of a.
// Returns a nil slice when no path can be found.
func (td TreeData2[T]) path(a, parent *T) []*T {
	if a == nil {
		return nil
	}
	aUID, ok := td.UID(a)
	if !ok {
		return nil
	}
	bUID, ok := td.UID(parent)
	if !ok {
		return nil
	}
	path := make([]*T, 0)
	if a == parent {
		return path
	}
	for {
		aUID = td.parents[aUID]
		if aUID == bUID {
			break
		}
		path = append(path, td.nodes[aUID])
	}
	slices.Reverse(path)
	return path
}

// Print prints a sub tree to the console.
// Nodes are expected to implement the stringer interface.
func (td TreeData2[T]) Print(node *T) {
	if uid, ok := td.UID(node); ok {
		td.print(uid, "", false)
		fmt.Println()
	}
}

func (td TreeData2[T]) print(uid widget.TreeNodeID, indent string, last bool) {
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

// LeafPaths returns a slice of paths to all leafs for a subtree.
// Nodes are expected to implement the stringer interface.
// The nil node represents the root.
func (td TreeData2[T]) LeafPaths(parent *T) [][]string {
	all := make([][]string, 0)
	for n := range td.All(parent) {
		if c, ok := td.ChildrenCount(n); ok && c == 0 {
			p := xslices.Map(td.path(n, parent), func(x *T) string {
				return fmt.Sprint(x)
			})
			p = append(p, fmt.Sprint(n))
			all = append(all, p)
		}
	}
	return all
}

// Size returns the total number of nodes in the tree excluding the root node.
func (td TreeData2[T]) Size() int {
	return len(td.nodes)
}

// String implements the stringer interface.
func (td TreeData2[T]) String() string {
	return fmt.Sprintf("{nodes %+v, children: %+v}", td.nodes, td.children)
}
