package datanodetree_test

import (
	"encoding/json"
	"testing"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/datanodetree"
	"github.com/stretchr/testify/assert"
)

type treeNode struct {
	Name   string
	IsLeaf bool
}

func (n treeNode) IsRoot() bool {
	return !n.IsLeaf
}

func (n treeNode) UID() widget.TreeNodeID {
	return n.Name
}

func (n treeNode) ToJSON() string {
	byt, err := json.Marshal(n)
	if err != nil {
		panic(err)
	}
	return string(byt)
}

func TestStringDataTree(t *testing.T) {
	t.Run("can render empty tree", func(t *testing.T) {
		tree := datanodetree.New[treeNode]()
		ids, values, err := tree.StringTree()
		if assert.NoError(t, err) {
			assert.Len(t, ids, 0)
			assert.Len(t, values, 0)
		}
	})
	t.Run("can render full tree", func(t *testing.T) {
		// given
		tree := datanodetree.New[treeNode]()
		alpha := treeNode{Name: "Alpha"}
		uid := tree.Add("", alpha)
		alpha1 := treeNode{Name: "Sub-Alpha-1"}
		tree.Add(uid, alpha1)
		alpha2 := treeNode{Name: "Sub-Alpha-2"}
		tree.Add(uid, alpha2)
		bravo := treeNode{Name: "Bravo"}
		tree.Add("", bravo)
		// when
		ids, values, err := tree.StringTree()
		// then
		if assert.NoError(t, err) {
			idsWant := map[string][]string{
				"":         {alpha.UID(), bravo.UID()},
				alpha.Name: {alpha1.UID(), alpha2.UID()},
			}
			assert.Equal(t, idsWant, ids)
			valuesWant := map[string]string{
				alpha.UID():  alpha.ToJSON(),
				alpha1.UID(): alpha1.ToJSON(),
				alpha2.UID(): alpha2.ToJSON(),
				bravo.UID():  bravo.ToJSON(),
			}
			assert.Equal(t, valuesWant, values)
		}
	})
	t.Run("should panic when trying to add with unknown parent UID", func(t *testing.T) {
		tree := datanodetree.New[treeNode]()
		assert.Panics(t, func() {
			tree.Add("unknown", treeNode{Name: "Dummy"})
		})
	})
	t.Run("should panic when trying to add another node with same UID", func(t *testing.T) {
		tree := datanodetree.New[treeNode]()
		tree.Add("", treeNode{Name: "Dummy"})
		assert.Panics(t, func() {
			tree.Add("", treeNode{Name: "Dummy"})
		})
	})
	t.Run("can retrieve node from data tree", func(t *testing.T) {
		// given
		tree := datanodetree.New[treeNode]()
		node := treeNode{Name: "Dummy"}
		tree.Add("", node)
		ids, values, err := tree.StringTree()
		if err != nil {
			t.Fatal(err)
		}
		st := binding.NewStringTree()
		if err := st.Set(ids, values); err != nil {
			t.Fatal(err)
		}
		// when
		node2, err := datanodetree.NodeFromBoundTree[treeNode](st, node.UID())
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, node, node2)
		}
	})
	t.Run("can retrieve node from data item", func(t *testing.T) {
		// given
		tree := datanodetree.New[treeNode]()
		node := treeNode{Name: "Dummy"}
		tree.Add("", node)
		ids, values, err := tree.StringTree()
		if err != nil {
			t.Fatal(err)
		}
		st := binding.NewStringTree()
		if err := st.Set(ids, values); err != nil {
			t.Fatal(err)
		}
		di, err := st.GetItem(node.UID())
		if err != nil {
			t.Fatal(err)
		}
		// when
		node2, err := datanodetree.NodeFromDataItem[treeNode](di)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, node, node2)
		}
	})
}
