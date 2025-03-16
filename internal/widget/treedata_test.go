package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type myNode struct {
	id string
	v  string
}

func (n myNode) UID() widget.TreeNodeID {
	return n.id
}

func TestFyneTree(t *testing.T) {
	t.Parallel()
	t.Run("can create tree", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		n1 := tree.MustAdd("", myNode{"1", "Alpha"})
		n11 := tree.MustAdd(n1, myNode{"11", "one"})
		n12 := tree.MustAdd(n1, myNode{"12", "two"})
		n2 := tree.MustAdd("", myNode{"2", "Bravo"})
		assert.Equal(t, 4, tree.Size())
		assert.Equal(t, []string{n11, n12}, tree.ChildUIDs(n1))
		assert.Len(t, tree.ChildUIDs(n2), 0)
		nodes := []struct {
			uid      string
			value    string
			isBranch bool
		}{
			{n1, "Alpha", true},
			{n11, "one", false},
			{n12, "two", false},
			{n2, "Bravo", false},
		}
		for _, x := range nodes {
			b := tree.MustNode(x.uid)
			assert.Equal(t, myNode{x.uid, x.value}, b)
			assert.Equal(t, x.isBranch, tree.IsBranch(x.uid))
		}
	})
}

func TestFyneTreeAdd(t *testing.T) {
	t.Parallel()
	t.Run("can add a node 1", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		n := myNode{"1", "Alpha"}
		uid, err := tree.Add("", n)
		if assert.NoError(t, err) {
			assert.Equal(t, n, tree.MustNode(uid))
		}
	})
	t.Run("can add a node 2", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		n := myNode{"1", "Alpha"}
		uid := tree.MustAdd("", n)
		assert.Equal(t, n, tree.MustNode(uid))
	})
	t.Run("should return error when node parent UID does not exist", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		_, err := tree.Add("invalid", myNode{"1", "Alpha"})
		assert.Error(t, err)
	})
	t.Run("should return error when node UID already exists", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		tree.MustAdd("", myNode{"1", "Alpha"})
		_, err := tree.Add("", myNode{"1", "Bravo"})
		assert.Error(t, err)
	})
	t.Run("should panic when node node can not be added", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		assert.Panics(t, func() {
			tree.MustAdd("invalid", myNode{"1", "Alpha"})
		})
	})
}

func TestFyneTreeValue(t *testing.T) {
	t.Parallel()
	t.Run("can return a node", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		n1 := myNode{"1", "Alpha"}
		uid := tree.MustAdd("", n1)
		n2 := tree.MustNode(uid)
		assert.Equal(t, n1, n2)
	})
	t.Run("should return node when it exists", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		n1 := myNode{"1", "Alpha"}
		uid := tree.MustAdd("", n1)
		n2, ok := tree.Node(uid)
		assert.True(t, ok)
		assert.Equal(t, n1, n2)

	})
	t.Run("should report when node does not exist", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		_, ok := tree.Node("invalid")
		assert.False(t, ok)
	})
	t.Run("should panic when node does not exist", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		assert.Panics(t, func() {
			tree.MustNode("invalid")

		})
	})
	t.Run("can return parent of a top node", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		uid := tree.MustAdd("", myNode{"1", "Alpha"})
		p, ok := tree.Parent(uid)
		assert.True(t, ok)
		assert.Equal(t, "", p)
	})
	t.Run("can return parent of a random node", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		uid1 := tree.MustAdd("", myNode{"1", "Alpha"})
		uid2 := tree.MustAdd(uid1, myNode{"2", "Bravo"})
		p, ok := tree.Parent(uid2)
		assert.True(t, ok)
		assert.Equal(t, uid1, p)
	})
	t.Run("can report when a parent does not exist", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		_, ok := tree.Parent("1")
		assert.False(t, ok)
	})
}

func TestFyneTreePath(t *testing.T) {
	t.Parallel()
	t.Run("should return path for an existing node", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		uid1 := tree.MustAdd("", myNode{"1", "Alpha"})
		uid2 := tree.MustAdd(uid1, myNode{"2", "Bravo"})
		uid3 := tree.MustAdd(uid2, myNode{"3", "Charlie"})
		p := tree.Path(uid3)
		assert.Equal(t, []widget.TreeNodeID{uid1, uid2}, p)
	})
	t.Run("should return empty array for root node", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		p := tree.Path("")
		assert.Equal(t, []widget.TreeNodeID{}, p)
	})
	t.Run("should return empty array for a top node", func(t *testing.T) {
		tree := iwidget.NewTreeData[myNode]()
		uid := tree.MustAdd("", myNode{"1", "Alpha"})
		p := tree.Path(uid)
		assert.Equal(t, []widget.TreeNodeID{}, p)
	})
}
