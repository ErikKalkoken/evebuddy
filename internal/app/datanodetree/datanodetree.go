package datanodetree

import (
	"encoding/json"
	"fmt"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// Any type which implements a UIDer can form a DataNodeTree.
type UIDer interface {
	// UID() returns a unique ID representing the node in the string tree.
	UID() widget.TreeNodeID
}

// DataNodeTree represents a tree of data nodes for making a string data tree widget.
// This type makes it more convenient to build correct trees, with a simpler API and automatic sanity checks.
type DataNodeTree[T UIDer] struct {
	ids    map[string][]string
	values map[string]T
}

// Returns a new DataNodeTree object.
func New[T UIDer]() DataNodeTree[T] {
	t := DataNodeTree[T]{
		values: make(map[string]T),
		ids:    make(map[string][]string),
	}
	return t
}

// Add adds a node to the tree and returns the UID.
// Nodes will be rendered in the same order they are added.
// Use "" as parentUID for adding nodes at the top level.
func (t DataNodeTree[T]) Add(parentUID string, node T) widget.TreeNodeID {
	if parentUID != "" {
		_, found := t.values[parentUID]
		if !found {
			panic(fmt.Sprintf("parent UID does not exist: %s", parentUID))
		}
	}
	uid := node.UID()
	_, found := t.values[uid]
	if found {
		panic(fmt.Sprintf("UID for this node already exists: %v", node))
	}
	t.ids[parentUID] = append(t.ids[parentUID], uid)
	t.values[uid] = node
	return uid
}

// StringTree returns the inputs for updating a bound Fyne StringTree.
func (t DataNodeTree[T]) StringTree() (map[string][]string, map[string]string, error) {
	values := make(map[string]string)
	for id, node := range t.values {
		v, err := objectToJSON(node)
		if err != nil {
			return nil, nil, err
		}
		values[id] = v
	}
	return t.ids, values, nil
}

// NodeFromBoundTree returns a data node from a bound string tree.
func NodeFromBoundTree[T any](data binding.StringTree, uid widget.TreeNodeID) (T, error) {
	var zero T
	v, err := data.GetValue(uid)
	if err != nil {
		return zero, fmt.Errorf("failed to get tree node: %w", err)
	}
	n, err := newObjectFromJSON[T](v)
	if err != nil {
		return zero, fmt.Errorf("failed to unmarshal tree node: %w", err)
	}
	return n, nil
}

// NodeFromDataItem returns a data node from a data item of a bound string tree.
func NodeFromDataItem[T any](di binding.DataItem) (T, error) {
	var zero T
	v, err := di.(binding.String).Get()
	if err != nil {
		return zero, err
	}
	n, err := newObjectFromJSON[T](v)
	if err != nil {
		return zero, err
	}
	return n, nil
}

// newObjectFromJSON returns a new object unmarshaled from a JSON string.
func newObjectFromJSON[T any](s string) (T, error) {
	var n T
	err := json.Unmarshal([]byte(s), &n)
	if err != nil {
		return n, err
	}
	return n, nil
}

// objectToJSON returns a JSON string marshaled from the given object.
func objectToJSON[T any](o T) (string, error) {
	s, err := json.Marshal(o)
	if err != nil {
		return "", err
	}
	return string(s), nil
}
