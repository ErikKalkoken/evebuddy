package treebuilder

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2/widget"
)

type FyneTree[T any] struct {
	mu     sync.RWMutex
	ids    map[widget.TreeNodeID][]widget.TreeNodeID
	values map[widget.TreeNodeID]T
}

func NewFyneTree[T any]() *FyneTree[T] {
	t := &FyneTree[T]{}
	t.initialize()
	return t
}

// Add adds a node safely to the tree and returns the UID again.
// It will return an error if a semantic check fails.
// Use "" as parentUID for adding nodes at the top level.
// Nodes will be rendered in the same order they are added.
func (t *FyneTree[T]) Add(parentUID widget.TreeNodeID, uid widget.TreeNodeID, value T) (widget.TreeNodeID, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
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

// Clear clears all nodes from the tree.
func (t *FyneTree[T]) Clear() {
	t.initialize()
}

func (t *FyneTree[T]) ChildUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ids[uid]
}

func (t *FyneTree[T]) IsBranch(uid widget.TreeNodeID) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, found := t.ids[uid]
	return found
}

// MustAdd is like Add, but panics if a semantic check fails.
func (t *FyneTree[T]) MustAdd(parentUID widget.TreeNodeID, uid widget.TreeNodeID, value T) widget.TreeNodeID {
	_, err := t.Add(parentUID, uid, value)
	if err != nil {
		panic(err)
	}
	return uid
}

func (t *FyneTree[T]) Value(uid widget.TreeNodeID) T {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.values[uid]
}

func (t *FyneTree[T]) initialize() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ids = make(map[widget.TreeNodeID][]widget.TreeNodeID)
	t.values = make(map[widget.TreeNodeID]T)
}
