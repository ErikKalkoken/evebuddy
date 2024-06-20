package stringdatatree

import (
	"encoding/json"
	"fmt"

	"fyne.io/fyne/v2/widget"
)

type dataNode interface {
	UID() widget.TreeNodeID
	IsRoot() bool
}

// StringDataTree represents a tree of data nodes for rendering with the tree widget.
type StringDataTree[T dataNode] struct {
	ids    map[string][]string
	values map[string]T
}

// Returns a new StringDataTree object.
func New[T dataNode]() StringDataTree[T] {
	ltd := StringDataTree[T]{
		values: make(map[string]T),
		ids:    make(map[string][]string),
	}
	return ltd
}

// Add adds a node to the tree and returns the UID.
func (ltd StringDataTree[T]) Add(parentUID string, node T) widget.TreeNodeID {
	if parentUID != "" {
		_, found := ltd.values[parentUID]
		if !found {
			panic(fmt.Sprintf("parent UID does not exist: %s", parentUID))
		}
	}
	uid := node.UID()
	_, found := ltd.values[uid]
	if found {
		panic(fmt.Sprintf("UID for this node already exists: %v", node))
	}
	ltd.ids[parentUID] = append(ltd.ids[parentUID], uid)
	ltd.values[uid] = node
	return uid
}

// StringTree returns the tree as input for a Fyne StringTree.
func (ltd StringDataTree[T]) StringTree() (map[string][]string, map[string]string, error) {
	values := make(map[string]string)
	for id, node := range ltd.values {
		v, err := objectToJSON(node)
		if err != nil {
			return nil, nil, err
		}
		values[id] = v
	}
	return ltd.ids, values, nil
}

// objectToJSON returns a JSON string marshaled from the given object.
func objectToJSON[T any](o T) (string, error) {
	s, err := json.Marshal(o)
	if err != nil {
		return "", err
	}
	return string(s), nil
}
