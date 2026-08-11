package xwidget

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

const (
	treeRootID widget.TreeNodeID = "" // UID of the root node in a Tree widget
)

// A TreeNode for [Tree].
type TreeNode interface {
	UID() widget.TreeNodeID
}

var (
	ErrInvalid       = errors.New("invalid operation")
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
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
type Tree[T TreeNode] struct {
	widget.Tree

	OnBranchClosedNode func(n *T) // Called when a Branch is closed
	OnBranchOpenedNode func(n *T) // Called when a Branch is opened
	OnSelectedNode     func(n *T) // Called when the given node is selected.
	OnUnselectedNode   func(n *T) // Called when the given node is unselected.

	td *TreeData[T]
}

// NewTree returns a new Tree2 object.
func NewTree[T TreeNode](
	create func(isBranch bool) fyne.CanvasObject,
	update func(n *T, isBranch bool, co fyne.CanvasObject),
) *Tree[T] {
	w := &Tree[T]{
		td: NewTreeData[T](),
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
	if w == nil {
		return
	}
	w.td.Clear()
	w.Refresh()
}

// Data returns the tree's data.
func (w *Tree[T]) Data() *TreeData[T] {
	if w == nil {
		return nil
	}
	return w.td
}

// Set replaces the tree's data, resets and refreshes it.
func (w *Tree[T]) Set(data *TreeData[T]) {
	if w == nil {
		fyne.LogError("Tree.Set: nil object", ErrInvalid)
		return
	}
	w.UnselectAll()
	w.CloseAllBranches()
	w.td = data
	w.Refresh()
	w.ScrollToTop()
}

// Wrappers below

// CloseBranchNode closes the branch of node n.
func (w *Tree[T]) CloseBranchNode(n *T) {
	if w == nil {
		return
	}
	w.CloseBranch(treeNodeUID(n))
}

// IsBranchOpenNode reports whether the given branch is expanded.
func (w *Tree[T]) IsBranchOpenNode(n *T) bool {
	if w == nil {
		return false
	}
	return w.IsBranchOpen(treeNodeUID(n))
}

// OpenBranchNode opens the branch of node n.
func (w *Tree[T]) OpenBranchNode(n *T) {
	if w == nil {
		return
	}
	w.OpenBranch(treeNodeUID(n))
}

// RefreshNode refreshes the given node.
func (w *Tree[T]) RefreshNode(n *T) {
	if w == nil {
		return
	}
	w.RefreshItem(treeNodeUID(n))
}

// ScrollToNode scrolls to node n.
func (w *Tree[T]) ScrollToNode(n *T) {
	if w == nil {
		return
	}
	w.ScrollTo(treeNodeUID(n))
}

// SelectNode marks node n to be selected.
func (w *Tree[T]) SelectNode(n *T) {
	if w == nil {
		return
	}
	w.Select(treeNodeUID(n))
}

// ToggleBranchNode flips the state of branch node n.
func (w *Tree[T]) ToggleBranchNode(n *T) {
	if w == nil {
		return
	}
	w.ToggleBranch(treeNodeUID(n))
}

// UnselectNode marks node n to be not selected.
func (w *Tree[T]) UnselectNode(n *T) {
	if w == nil {
		return
	}
	w.Unselect(treeNodeUID(n))
}

// TreeData represents the data for rendering a [Tree] widget
// and provides operations for querying and modifying the tree.
// It is optimized for fast widget rendering.
//
// A tree is constructed by adding nodes to a virtual root node.
// The root node always exists and is represented by the nil node.
type TreeData[T TreeNode] struct {
	children map[widget.TreeNodeID][]widget.TreeNodeID
	isBranch map[widget.TreeNodeID]bool
	nodes    map[widget.TreeNodeID]*T
	parents  map[widget.TreeNodeID]widget.TreeNodeID
}

// NewTreeData returns a new TreeData object.
func NewTreeData[T TreeNode]() *TreeData[T] {
	td := &TreeData[T]{}
	td.init()
	return td
}

func (td *TreeData[T]) init() {
	td.children = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	td.isBranch = make(map[widget.TreeNodeID]bool)
	td.nodes = make(map[widget.TreeNodeID]*T)
	td.parents = make(map[widget.TreeNodeID]widget.TreeNodeID)
}

// Add adds a node n to parent.
// //
// The order in which nodes are added is preserved.
// Add performs sanity checks to ensure the resulting tree structure is valid
// and returns an error when a problem was found.
//
// Nodes can be added to the root by providing nil for parent.
func (td *TreeData[T]) Add(parent *T, n *T, isBranch bool) error {
	if td == nil || n == nil {
		return ErrInvalid
	}
	if td.children == nil {
		td.init() // init zero value
	}
	parentUID, ok := td.UID(parent)
	if !ok {
		return fmt.Errorf("TreeData.Add: parent not found: %w", ErrNotFound)
	}
	if parentUID != treeRootID && !td.isBranch[parentUID] {
		return fmt.Errorf("TreeData.Add: parent must be a branch: %w", ErrInvalid)
	}
	uid := treeNodeUID(n)
	if _, found := td.nodes[uid]; found {
		return fmt.Errorf("TreeData.Add: node %s: %w", uid, ErrAlreadyExists)
	}
	td.children[parentUID] = append(td.children[parentUID], uid)
	td.nodes[uid] = n
	td.parents[uid] = parentUID
	td.isBranch[uid] = isBranch
	return nil
}

// Children returns a new slice with the direct children of node parent.
// The children are returned in the same order as they were added.
// When parent was not found a nil slice is returned.
func (td *TreeData[T]) Children(n *T) []*T {
	if td == nil {
		return nil
	}
	uid, ok := td.UID(n)
	if !ok {
		return nil
	}
	var nodes []*T
	for _, id := range td.children[uid] {
		nodes = append(nodes, td.nodes[id])
	}
	return nodes
}

// ChildrenCount returns the number of direct children of node parent.
// It returns 0 when the node is not found.
func (td *TreeData[T]) ChildrenCount(n *T) int {
	if td == nil {
		return 0
	}
	uid, found := td.UID(n)
	if !found {
		return 0
	}
	return len(td.children[uid])
}

// Clear removes all nodes.
func (td *TreeData[T]) Clear() {
	if td == nil {
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
func (td *TreeData[T]) Clone() *TreeData[T] {
	if td == nil {
		return nil
	}
	t2 := &TreeData[T]{
		children: make(map[widget.TreeNodeID][]widget.TreeNodeID, len(td.children)),
		isBranch: maps.Clone(td.isBranch),
		nodes:    maps.Clone(td.nodes),
		parents:  maps.Clone(td.parents),
	}
	for k, v := range td.children {
		t2.children[k] = slices.Clone(v)
	}
	return t2
}

// Delete deletes the given nodes including their subtrees.
// It will return an error if a node can not be deleted.
// The root node can not be removed.
func (td *TreeData[T]) Delete(nodes ...*T) error {
	if td == nil {
		return fmt.Errorf("TreeData.Delete: nil object: %w", ErrInvalid)
	}
	for _, n := range nodes {
		if n == nil {
			return fmt.Errorf("TreeData.Delete: can not remove root node: %w", ErrInvalid)
		}
		uid, found := td.UID(n)
		if !found {
			return fmt.Errorf("TreeData.Delete: %w", ErrNotFound)
		}
		td.delete(uid)
	}
	return nil
}

func (td *TreeData[T]) delete(uid widget.TreeNodeID) {
	if td == nil {
		return
	}
	s, found := td.children[uid]
	if found {
		s2 := slices.Clone(s)
		for _, n := range s2 {
			if n == treeRootID {
				fyne.LogError("TreeData.delete: root ID found in children: "+uid, ErrInvalid)
				return
			}
			td.delete(n)
		}
		delete(td.children, uid)
	}
	parent, found := td.parents[uid]
	if !found {
		fyne.LogError("TreeData.delete: Parent not found for UID: "+uid, ErrInvalid)
		return
	}
	td.children[parent] = slices.DeleteFunc(td.children[parent], func(x widget.TreeNodeID) bool {
		return x == uid
	})
	delete(td.parents, uid)
	delete(td.nodes, uid)
	delete(td.isBranch, uid)
}

// DeleteChildrenFunc removes any nodes from parent for which del returns true.
// It does nothing when parent is not found.
func (td *TreeData[T]) DeleteChildrenFunc(parent *T, del func(n *T) bool) {
	if td == nil {
		return
	}
	uid, ok := td.UID(parent)
	if !ok {
		return
	}
	var toDelete []widget.TreeNodeID
	for _, childUID := range td.children[uid] {
		if del(td.nodes[childUID]) {
			toDelete = append(toDelete, childUID)
		}
	}

	for _, childUID := range toDelete {
		td.delete(childUID)
	}
}

// Exists reports whether a node exists.
// Nil will also return represents the root node and will also return true.
func (td *TreeData[T]) Exists(n *T) bool {
	_, ok := td.UID(n)
	return ok
}

// IsEmpty reports whether the tree has any nodes (other then the root node).
func (td *TreeData[T]) IsEmpty() bool {
	return td == nil || len(td.nodes) == 0
}

func (td *TreeData[T]) IsBranch(n *T) bool {
	if td == nil {
		return false
	}
	if n == nil {
		return true // root is always a branch
	}
	if uid, ok := td.UID(n); ok {
		return td.isBranch[uid]
	}
	return false
}

// Node returns a node by UID and reports whether it was found.
// The root node will be returned as nil.
func (td *TreeData[T]) Node(uid widget.TreeNodeID) (*T, bool) {
	if td == nil {
		return nil, false
	}
	if uid == treeRootID {
		return nil, true
	}
	n, ok := td.nodes[uid]
	return n, ok
}

// Parent returns the parent of a node and reports whether the operation succeeded.
func (td *TreeData[T]) Parent(n *T) (*T, bool) {
	if td == nil {
		return nil, false
	}
	uid, ok := td.UID(n)
	if !ok {
		return nil, false
	}
	parent, ok2 := td.parents[uid]
	if !ok2 {
		return nil, false
	}
	return td.nodes[parent], true
}

// Path returns the path from parent to n.
// The path includes parent (except root) and n.
// Parent must be an ancestor of n or nil for the root node.
// Returns a nil slice when no path can be found.
func (td *TreeData[T]) Path(parent, n *T) []*T {
	if td == nil || n == nil {
		return nil
	}
	a, ok := td.UID(n)
	if !ok {
		return nil
	}
	b, ok := td.UID(parent)
	if !ok {
		return nil
	}
	path := []*T{n}
	for {
		a = td.parents[a]
		if a != treeRootID {
			path = append(path, td.nodes[a])
		}
		if a == b {
			break
		}
		if a == treeRootID {
			return nil // parent was not an ancestor
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
func (td *TreeData[T]) AllPaths(parent *T, stringify func(*T) string) [][]string {
	if td == nil {
		return nil
	}
	var all [][]string
	td.Walk(parent, func(n *T) bool {
		if td.ChildrenCount(n) == 0 {
			var p []string
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
func (td *TreeData[T]) Print(n *T, stringify func(*T) string) {
	if td == nil {
		fyne.LogError("TreeData.Print: nil object: ", ErrInvalid)
		return
	}

	uid, ok := td.UID(n)
	if !ok {
		fyne.LogError("TreeData.Print: node not found: ", ErrInvalid)
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
		children := td.children[uid]
		for i, id := range td.children[uid] {
			printTreeData(id, indent, i == len(children)-1)
		}
	}

	printTreeData(uid, "", false)
	fmt.Println()
}

// SortChildrenFunc sorts the direct children of parent in ascending order
// as determined by the cmp function.
// It does nothing when parent is not found.
func (td *TreeData[T]) SortChildrenFunc(parent *T, cmp func(a *T, b *T) int) {
	if td == nil {
		return
	}
	uid, ok := td.UID(parent)
	if !ok {
		return
	}
	slices.SortFunc(td.children[uid], func(a, b widget.TreeNodeID) int {
		return cmp(td.nodes[a], td.nodes[b])
	})
}

func (td *TreeData[T]) SetBranch(n *T, isBranch bool) error {
	if td == nil {
		return fmt.Errorf("SetBranch: nil object: %w", ErrInvalid)
	}
	if n == nil {
		return fmt.Errorf("SetBranch: can not set root: %w", ErrInvalid)
	}
	uid, ok := td.UID(n)
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
func (td *TreeData[T]) Size() int {
	if td == nil {
		return 0
	}
	return len(td.nodes)
}

// UID returns the UID of node n when it exists and reports whether it exists.
// Nil returns the root node.
func (td *TreeData[T]) UID(n *T) (widget.TreeNodeID, bool) {
	if td == nil {
		return "", false
	}
	if n == nil {
		return treeRootID, true
	}
	uid := treeNodeUID(n)
	_, ok := td.nodes[uid]
	if !ok {
		return "", false
	}
	return uid, true
}

// Walk walks the sub tree of parent, calling f for each node in the tree,
// including parent (except the root).
// It continues walking until all nodes have been visited or f returns false.
//
// The nodes are walked in depth first order.
// Walk starts at the root when parent is nil.
// Walk does nothing if parent is not found.
func (td *TreeData[T]) Walk(parent *T, f func(n *T) bool) {
	if td == nil {
		return
	}
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

// treeNodeUID returns the UID for a tree node.
// It return the root node for nil.
func treeNodeUID[T TreeNode](n *T) widget.TreeNodeID {
	if n == nil {
		return treeRootID
	}
	return (*n).UID()
}
