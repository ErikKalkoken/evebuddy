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
	treeRootID widget.TreeNodeID = "" // UID of the root node in a Tree widget
)

var (
	ErrInvalid  = errors.New("invalid operation")
	ErrNotFound = errors.New("not found")
)

// Tree is an extension of Fyne's Tree widget that allows to create trees from nodes.
// A node can be any struct.
//
// It also provides node based alternatives for all tree methods that operate on UIDs.
// The alternatives will ignore calls with an non-existing node just like the originals.
//
// The tree structure is defined through a [TreeData] object.
// This allows creating the structure of a tree independently from the rendered tree.
//
// Do not set any of the original callbacks as this would disable the functionality
// of this widget.
type Tree[T any] struct {
	widget.Tree

	OnBranchClosedNode func(n *T) // Called when a Branch is closed
	OnBranchOpenedNode func(n *T) // Called when a Branch is opened
	OnSelectedNode     func(n *T) // Called when the given node is selected.
	OnUnselectedNode   func(n *T) // Called when the given node is unselected.

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
	w.Root = treeRootID
	w.ChildUIDs = func(uid widget.TreeNodeID) []widget.TreeNodeID {
		return w.td.children[uid]
	}
	w.IsBranch = func(uid widget.TreeNodeID) bool {
		if uid == treeRootID {
			return true
		}
		return w.td.isBranch[uid]
	}
	w.CreateNode = create
	w.UpdateNode = func(uid widget.TreeNodeID, isBranch bool, co fyne.CanvasObject) {
		if n, ok := w.td.Node(uid); ok {
			update(n, isBranch, co)
		}
	}
	callWhenExists := func(f func(n *T), uid widget.TreeNodeID) {
		if f != nil {
			if n, ok := w.td.nodes[uid]; ok {
				f(n)
			}
		}
	}
	w.OnBranchClosed = func(uid widget.TreeNodeID) {
		callWhenExists(w.OnBranchClosedNode, uid)
	}
	w.OnBranchOpened = func(uid widget.TreeNodeID) {
		callWhenExists(w.OnBranchOpenedNode, uid)
	}
	w.OnSelected = func(uid widget.TreeNodeID) {
		callWhenExists(w.OnSelectedNode, uid)
	}
	w.OnUnselected = func(uid widget.TreeNodeID) {
		callWhenExists(w.OnUnselectedNode, uid)
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

// Wrappers below

// CloseBranchNode closes the branch of node n.
func (w *Tree[T]) CloseBranchNode(n *T) {
	w.callWhenFound(n, w.CloseBranch)
}

// IsBranchOpenNode reports whether the given branch is expanded.
func (w *Tree[T]) IsBranchOpenNode(n *T) bool {
	if uid, ok := w.td.uidLookup[n]; ok {
		return w.IsBranchOpen(uid)
	}
	return false
}

// OpenBranchNode opens the branch of node n.
func (w *Tree[T]) OpenBranchNode(n *T) {
	w.callWhenFound(n, w.OpenBranch)
}

// RefreshNode refreshes the given node.
func (w *Tree[T]) RefreshNode(n *T) {
	w.callWhenFound(n, w.RefreshItem)
}

// ScrollToNode scrolls to node n.
func (w *Tree[T]) ScrollToNode(n *T) {
	w.callWhenFound(n, w.ScrollTo)
}

// SelectNode marks node n to be selected.
func (w *Tree[T]) SelectNode(n *T) {
	w.callWhenFound(n, w.Select)
}

// ToggleBranchNode flips the state of branch node n.
func (w *Tree[T]) ToggleBranchNode(n *T) {
	w.callWhenFound(n, w.ToggleBranch)
}

// UnselectNode marks node n to be not selected.
func (w *Tree[T]) UnselectNode(n *T) {
	w.callWhenFound(n, w.Unselect)
}

func (w *Tree[T]) callWhenFound(n *T, f func(widget.TreeNodeID)) {
	if uid, ok := w.td.uidLookup[n]; ok {
		f(uid)
	}
}

// TreeData represents the data for rendering a [Tree] widget
// and provides operations for querying and modifying the tree.
//
// A tree is constructed by adding nodes to a virtual root node.
// The root node always exists and is represented by the nil node.
//
// The zero value is an empty tree ready to use.
type TreeData[T any] struct {
	children  map[widget.TreeNodeID][]widget.TreeNodeID
	isBranch  map[widget.TreeNodeID]bool
	lastID    int
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
	td.lastID = 0
	td.isBranch = make(map[widget.TreeNodeID]bool)
	td.nodes = make(map[widget.TreeNodeID]*T)
	td.parents = make(map[widget.TreeNodeID]widget.TreeNodeID)
	td.uidLookup = make(map[*T]widget.TreeNodeID)
}

// Add adds a node to parent.
//
// The order in which nodes are added is preserved.
// Add performs sanity checks to ensure the resulting tree structure is valid
// and returns an error when a problem was found.
//
// Nodes can be added to the root by providing nil for parent.
func (td *TreeData[T]) Add(parent *T, node *T, isBranch bool) error {
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
	if parentUID != treeRootID && !td.isBranch[parentUID] {
		return fmt.Errorf("can not add to non-branch: %w", ErrInvalid)
	}
	td.lastID++
	uid := strconv.Itoa(td.lastID)
	td.children[parentUID] = append(td.children[parentUID], uid)
	td.nodes[uid] = node
	td.parents[uid] = parentUID
	td.uidLookup[node] = uid
	td.isBranch[uid] = isBranch
	return nil
}

// Children returns a new slice with the direct children of node parent.
// The children are returned in the same order as they were added.
// When parent was not found a nil slice is returned.
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

// ChildrenCount returns the number of direct children of node parent
// and reports whether the node was found.
func (td TreeData[T]) ChildrenCount(parent *T) (int, bool) {
	uid, found := td.UID(parent)
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

// Clone returns a shallow copy of the tree data object.
// A clone can be used to modify the structure of a tree separately
// and then update the tree widget later in one operation.
func (td TreeData[T]) Clone() TreeData[T] {
	t2 := TreeData[T]{
		children:  make(map[widget.TreeNodeID][]widget.TreeNodeID),
		isBranch:  maps.Clone(td.isBranch),
		lastID:    td.lastID,
		nodes:     maps.Clone(td.nodes),
		parents:   maps.Clone(td.parents),
		uidLookup: maps.Clone(td.uidLookup),
	}
	for k, v := range td.children {
		t2.children[k] = slices.Clone(v)
	}
	return t2
}

// Delete deletes the given nodes including their subtrees.
// It will return an error if a node can not be deleted.
// The root node can not be removed.
func (td TreeData[T]) Delete(node ...*T) error {
	for _, n := range node {
		if n == nil {
			return fmt.Errorf("Delete: can not remove root node: %w", ErrInvalid)
		}
		uid, found := td.uidLookup[n]
		if !found {
			return fmt.Errorf("Delete: uid %s: %w", uid, ErrNotFound)
		}
		td.delete(uid)
	}
	return nil
}

func (td TreeData[T]) delete(uid widget.TreeNodeID) {
	s, found := td.children[uid]
	if found {
		s2 := slices.Clone(s)
		for _, n := range s2 {
			if n == treeRootID {
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

// DeleteChildrenFunc removes any nodes from parent for which del returns true.
// It does nothing when parent is not found.
func (td TreeData[T]) DeleteChildrenFunc(parent *T, del func(n *T) bool) {
	uid, ok := td.UID(parent)
	if !ok {
		return
	}
	td.children[uid] = slices.DeleteFunc(td.children[uid], func(n widget.TreeNodeID) bool {
		return del(td.nodes[n])
	})
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

func (td TreeData[T]) IsBranch(node *T) bool {
	if node == nil {
		return true // root is always a branch
	}
	if uid, ok := td.uidLookup[node]; ok {
		return td.isBranch[uid]
	}
	return false
}

// Node returns a node by UID and reports whether it was found.
// The root node will be returned as nil.
func (td TreeData[T]) Node(uid widget.TreeNodeID) (node *T, ok bool) {
	if uid == treeRootID {
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
		if aUID != treeRootID {
			path = append(path, td.nodes[aUID])
		}
		if aUID == bUID {
			break
		}
	}
	slices.Reverse(path)
	return path
}

// AllPaths returns a slice of node paths from parent to all leafs.
// Each node is converted to a string using stringify.
// This is a type of linearization and can be useful for comparing trees in tests.
//
// Will return all paths from root when parent is nil.
func (td TreeData[T]) AllPaths(parent *T, stringify func(*T) string) [][]string {
	all := make([][]string, 0)
	td.Walk(parent, func(n *T) bool {
		if k, ok := td.ChildrenCount(n); ok && k == 0 {
			p := make([]string, 0)
			for _, x := range td.Path(parent, n) {
				p = append(p, stringify(x))
			}
			all = append(all, p)
		}
		return true
	})
	return all
}

// Print prints a sub tree of node n to the console using stringify for each node.
// This can be useful to visualize a tree for debugging.
func (td TreeData[T]) Print(n *T, stringify func(*T) string) {
	uid, ok := td.UID(n)
	if !ok {
		return
	}

	var printTreeData func(widget.TreeNodeID, string, bool)
	printTreeData = func(uid widget.TreeNodeID, indent string, last bool) {
		var s string
		if uid == treeRootID {
			s = "ROOT"
		} else {
			n, _ := td.Node(uid)
			s = stringify(n)
		}
		fmt.Println(indent + "+- " + s)
		if last {
			indent += "   "
		} else {
			indent += "|  "
		}
		for _, id := range td.children[uid] {
			printTreeData(id, indent, len(td.children[id]) == 0)
		}
	}

	printTreeData(uid, "", false)
	fmt.Println()
}

// SortChildrenFunc sorts the direct children of parent in ascending order
// as determined by the cmp function.
// It does nothing when parent is not found.
func (td TreeData[T]) SortChildrenFunc(parent *T, cmp func(a *T, b *T) int) {
	uid, ok := td.UID(parent)
	if !ok {
		return
	}
	slices.SortFunc(td.children[uid], func(a, b widget.TreeNodeID) int {
		return cmp(td.nodes[a], td.nodes[b])
	})
}

func (td TreeData[T]) SetBranch(node *T, isBranch bool) error {
	if node == nil {
		return fmt.Errorf("SetBranch: can not set root: %w", ErrInvalid)
	}
	uid, ok := td.uidLookup[node]
	if !ok {
		return fmt.Errorf("SetBranch: %w", ErrNotFound)
	}
	if !isBranch && len(td.children[uid]) > 0 {
		return fmt.Errorf("SetBranch: node with children can not be leaf %w", ErrInvalid)
	}
	td.isBranch[uid] = isBranch
	return nil
}

// Size returns the number of nodes in the tree excluding the virtual root node.
func (td TreeData[T]) Size() int {
	return len(td.nodes)
}

// UID returns the UID for a node and reports whether it was found.
// Nil represents the root node and is valid.
func (td TreeData[T]) UID(node *T) (uid widget.TreeNodeID, ok bool) {
	if node == nil {
		return treeRootID, true
	}
	uid, ok = td.uidLookup[node]
	return
}

// Walk walks the sub tree of parent, calling f for each node in the tree,
// including parent (except the root).
// It continues walking until all nodes have been visited or f returns false.
//
// The nodes are walked in depth first order.
// Walk starts at the root when parent is nil.
// Walk does nothing if parent is not found.
func (td TreeData[T]) Walk(parent *T, f func(n *T) bool) {
	var traverse func(widget.TreeNodeID) bool
	traverse = func(curr widget.TreeNodeID) bool {
		if curr != treeRootID {
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
