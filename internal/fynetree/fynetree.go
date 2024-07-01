package fynetree

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2/widget"
)

// FyneTree represents the data for rendering a tree with a Fyne widget.
//
// FyneTree allows constructing and updating trees safely by performing semantic checks and being thread safe.
// It can be plugged-in directly into the functions of a regular Fyne tree widget.
// Nodes can be of any type.
// Node IDs are generated automatically.
type FyneTree[T any] struct {
	mu     sync.RWMutex
	ids    map[widget.TreeNodeID][]widget.TreeNodeID
	values map[widget.TreeNodeID]T
	id     int
}

// New returns a new FyneTree object.
func New[T any]() *FyneTree[T] {
	t := &FyneTree[T]{}
	t.initialize()
	return t
}

// Add adds a node safely and returns it's UID.
// Use "" as parentUID for adding nodes at the top level.
// Nodes will be rendered in the same order they are added.
func (t *FyneTree[T]) Add(parentUID widget.TreeNodeID, value T) (widget.TreeNodeID, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if parentUID != "" {
		_, found := t.values[parentUID]
		if !found {
			return "", fmt.Errorf("parent node does not exist: %s", parentUID)
		}
	}
	t.id++
	uid := fmt.Sprintf("%s-%d", parentUID, t.id)
	_, found := t.values[uid]
	if found {
		panic(fmt.Sprintf("this node already exists: %v", uid))
	}
	t.ids[parentUID] = append(t.ids[parentUID], uid)
	t.values[uid] = value
	return uid, nil
}

// Clear clears all nodes from the tree.
func (t *FyneTree[T]) Clear() {
	t.initialize()
}

// ChildUIDs returns the child UIDs of a node.
func (t *FyneTree[T]) ChildUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ids[uid]
}

// IsBranch reports wether a node is a branch.
func (t *FyneTree[T]) IsBranch(uid widget.TreeNodeID) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, found := t.ids[uid]
	return found
}

// MustAdd is like Add, but panics if a semantic check fails.
func (t *FyneTree[T]) MustAdd(parentUID widget.TreeNodeID, value T) widget.TreeNodeID {
	uid, err := t.Add(parentUID, value)
	if err != nil {
		panic(err)
	}
	return uid
}

// Value returns the value of a node.
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
	t.id = 0
}
