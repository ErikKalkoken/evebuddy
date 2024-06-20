package ui

import (
	"fmt"

	"fyne.io/fyne/v2/widget"
)

type dataNode interface {
	UID() widget.TreeNodeID
	isRoot() bool
}

// stringDataTree represents a tree of data nodes for rendering with the tree widget.
type stringDataTree[T dataNode] struct {
	ids    map[string][]string
	values map[string]T
}

func newStringDataTree[T dataNode]() stringDataTree[T] {
	ltd := stringDataTree[T]{
		values: make(map[string]T),
		ids:    make(map[string][]string),
	}
	return ltd
}

func (ltd stringDataTree[T]) add(parentUID string, node T) string {
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

func (ltd stringDataTree[T]) stringTree() (map[string][]string, map[string]string, error) {
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
