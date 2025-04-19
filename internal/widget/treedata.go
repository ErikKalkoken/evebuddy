// Package fynetree contains a type that makes using Fyne tree widgets easier.
package widget

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2/widget"
)

const (
	RootUID widget.TreeNodeID = ""
)

// TreeNode is the interface for a node in a Fyne tree.
type TreeNode interface {
	// UID returns a unique ID for a node.
	UID() widget.TreeNodeID
}

// TreeData is a type that holds all data needed to render a Fyne tree widget.
//
// It is designed to make it easier to construct tree data structures for a widget
// by hiding some of the complexity behind a simplery API and by performing sanity checks.
//
// Trees are constructed by adding nodes, which can contain any data
// as long as it complies with the [TreeNode] interface.
//
// Please note that nodes that have child nodes are always reported as branches.
// This means there can not be any empty branch nodes.
//
// This type is not thread safe.
type TreeData[T TreeNode] struct {
	ids     map[widget.TreeNodeID][]widget.TreeNodeID
	parents map[widget.TreeNodeID]widget.TreeNodeID
	nodes   map[widget.TreeNodeID]T
}

// NewTreeData returns a new TreeData object.
func NewTreeData[T TreeNode]() *TreeData[T] {
	t := &TreeData[T]{
		ids:     make(map[widget.TreeNodeID][]widget.TreeNodeID),
		parents: make(map[widget.TreeNodeID]widget.TreeNodeID),
		nodes:   make(map[widget.TreeNodeID]T),
	}
	return t
}

// Add adds a node safely. It returns it's UID or an error if the node can not be added.
//
// Use [RootUID] as parentUID for adding nodes at the top level.
// Nodes will be rendered in the same order as they are added.
func (t *TreeData[T]) Add(parentUID widget.TreeNodeID, node T) (widget.TreeNodeID, error) {
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

// ChildUIDs returns the child UIDs of a node.
func (t *TreeData[T]) ChildUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	if t == nil {
		return make([]widget.TreeNodeID, 0)
	}
	return t.ids[uid]
}

// FIXME: Method does not return full tree
func (t *TreeData[T]) Flat() []T {
	var s []T
	uid := ""
	for _, id := range t.ChildUIDs(uid) {
		s = append(s, t.MustNode(id))
	}
	return s
}

// IsBranch reports wether a node is a branch.
func (t *TreeData[T]) IsBranch(uid widget.TreeNodeID) bool {
	if t == nil {
		return false
	}
	_, found := t.ids[uid]
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
func (t *TreeData[T]) MustNode(uid widget.TreeNodeID) T {
	v, ok := t.Node(uid)
	if !ok {
		panic(fmt.Sprintf("node %s does not exist", uid))
	}
	return v
}

// Path returns the UIDs of nodes between a given node and the root.
func (t *TreeData[T]) Path(uid widget.TreeNodeID) []widget.TreeNodeID {
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
func (t *TreeData[T]) Parent(uid widget.TreeNodeID) (parent widget.TreeNodeID, ok bool) {
	parent, ok = t.parents[uid]
	return
}

// Size returns the number of nodes in the tree
func (t *TreeData[T]) Size() int {
	return len(t.nodes)
}

// Node returns a node by UID and reports whether it exists.
//
// Note that when using this method with a Fyne widget it is possible,
// that a UID forwarded by the widget no longer exists due to race conditions.
// It is therefore recommended to always check the ok value.
func (t *TreeData[T]) Node(uid widget.TreeNodeID) (node T, ok bool) {
	node, ok = t.node(uid)
	return
}

// node returns a node by UID and reports whether it exists.
func (t *TreeData[T]) node(uid widget.TreeNodeID) (T, bool) {
	v, ok := t.nodes[uid]
	return v, ok
}
