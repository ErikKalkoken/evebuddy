// Package fynetree contains a type that makes using Fyne tree widgets easier.
package fynetree

import (
	"fmt"

	"fyne.io/fyne/v2/widget"
)

// FyneTree is a type that holds all data needed to render a Fyne tree widget.
//
// It is designed to make it easier and safer to construct and update tree widgets
// and is safe to use with go routines.
//
// It's method are designed to be used directly inside the functions for creating and updating a fyne tree.
//
// Nodes can be of any type.
// Nodes that have child nodes are reported as branches. Note that this means there can not be any empty branch nodes.
type FyneTree[T any] struct {
	ids    map[widget.TreeNodeID][]widget.TreeNodeID
	values map[widget.TreeNodeID]T
}

// New returns a new FyneTree object.
func New[T any]() *FyneTree[T] {
	t := &FyneTree[T]{}
	t.initialize()
	return t
}

// Add adds a node safely. It returns it's UID or an error if the node can not be added.
//
// Use "" as parentUID for adding nodes at the top level.
// Nodes will be rendered in the same order as they are added.
func (t *FyneTree[T]) Add(parentUID widget.TreeNodeID, uid widget.TreeNodeID, value T) (widget.TreeNodeID, error) {
	if parentUID != "" {
		_, found := t.values[parentUID]
		if !found {
			return "", fmt.Errorf("parent node does not exist: %s", parentUID)
		}
	}
	_, found := t.values[uid]
	if found {
		return "", fmt.Errorf("this node already exists: %v", uid)
	}
	t.ids[parentUID] = append(t.ids[parentUID], uid)
	t.values[uid] = value
	return uid, nil
}

// MustAdd is like Add, but panics if adding fails.
func (t *FyneTree[T]) MustAdd(parentUID widget.TreeNodeID, uid widget.TreeNodeID, value T) widget.TreeNodeID {
	uid, err := t.Add(parentUID, uid, value)
	if err != nil {
		panic(err)
	}
	return uid
}

// ChildUIDs returns the child UIDs of a node.
func (t *FyneTree[T]) ChildUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	return t.ids[uid]
}

// IsBranch reports wether a node is a branch.
func (t *FyneTree[T]) IsBranch(uid widget.TreeNodeID) bool {
	_, found := t.ids[uid]
	return found
}

// Size returns the number of nodes in the tree
func (t *FyneTree[T]) Size() int {
	return len(t.values)
}

// Value returns the value of a node or an error if it does not exist.
//
// Note that when using this method inside a Fyne widget function it is possible,
// that a UID forwarded by the widget no longer exists.
// This error case should be handled gracefully.
func (t *FyneTree[T]) Value(uid widget.TreeNodeID) (T, error) {
	var zero T
	v, ok := t.value(uid)
	if !ok {
		return zero, fmt.Errorf("node does not exist: %s", uid)
	}
	return v, nil
}

// MustValue returns the value of a node or panics if the node does not exist.
// This method mainly exists to simplify test code and should not be used in production code.
func (t *FyneTree[T]) MustValue(uid widget.TreeNodeID) T {
	v, err := t.Value(uid)
	if err != nil {
		panic(err)
	}
	return v
}

// Value returns the value of a node or a fallback value.
func (t *FyneTree[T]) ValueWithFallback(uid widget.TreeNodeID, fallback T) T {
	v, ok := t.value(uid)
	if !ok {
		return fallback
	}
	return v
}

// Value returns the value of a node and a test flag reporting wether the node exists.
func (t *FyneTree[T]) value(uid widget.TreeNodeID) (T, bool) {
	v, ok := t.values[uid]
	return v, ok
}

func (t *FyneTree[T]) initialize() {
	t.ids = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	t.values = make(map[widget.TreeNodeID]T)
}
