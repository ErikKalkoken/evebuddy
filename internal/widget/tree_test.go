package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type MyNode struct {
	ID    string
	Value string
}

func (n MyNode) UID() widget.TreeNodeID {
	return n.ID
}

func TestTreeNodes_Create(t *testing.T) {
	t.Run("can create tree", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		n1 := tree.MustAdd("", MyNode{"1", "Alpha"})
		n11 := tree.MustAdd(n1, MyNode{"11", "one"})
		n12 := tree.MustAdd(n1, MyNode{"12", "two"})
		n2 := tree.MustAdd("", MyNode{"2", "Bravo"})
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
			assert.Equal(t, MyNode{x.uid, x.value}, b)
			assert.Equal(t, x.isBranch, tree.IsBranch(x.uid))
		}
	})
}

func TestTreeNodes_Add(t *testing.T) {
	t.Run("can add a node", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		n := MyNode{"1", "Alpha"}
		uid, err := tree.Add("", n)
		if assert.NoError(t, err) {
			assert.Equal(t, n, tree.MustNode(uid))
		}
	})
	t.Run("should return error when node parent UID does not exist", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		_, err := tree.Add("invalid", MyNode{"1", "Alpha"})
		assert.Error(t, err)
	})
	t.Run("can add a node to zero value", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		n := MyNode{"1", "Alpha"}
		uid, err := tree.Add("", n)
		if assert.NoError(t, err) {
			assert.Equal(t, n, tree.MustNode(uid))
		}
	})
	t.Run("should return error when trying to add to a nil pointer", func(t *testing.T) {
		var tree *iwidget.TreeNodes[MyNode]
		_, err := tree.Add("", MyNode{"1", "Alpha"})
		assert.ErrorIs(t, err, iwidget.ErrUndefined)
	})
}

func TestTreeNodes_MustAdd(t *testing.T) {
	t.Run("can add a node", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		n := MyNode{"1", "Alpha"}
		uid := tree.MustAdd("", n)
		assert.Equal(t, n, tree.MustNode(uid))
	})
	t.Run("should return error when node UID already exists", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		tree.MustAdd("", MyNode{"1", "Alpha"})
		_, err := tree.Add("", MyNode{"1", "Bravo"})
		assert.Error(t, err)
	})
	t.Run("should return error when node UID is root", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		tree.MustAdd("", MyNode{"1", "Alpha"})
		_, err := tree.Add("", MyNode{"", "Bravo"})
		assert.Error(t, err)
	})
	t.Run("should panic when node node can not be added", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		assert.Panics(t, func() {
			tree.MustAdd("invalid", MyNode{"1", "Alpha"})
		})
	})
	t.Run("can add a node to zero value", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		n := MyNode{"1", "Alpha"}
		uid := tree.MustAdd("", n)
		assert.Equal(t, n, tree.MustNode(uid))
	})
}

func TestTreeNodes_Node(t *testing.T) {
	t.Run("should return node when it exists", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		n1 := MyNode{"1", "Alpha"}
		uid := tree.MustAdd("", n1)
		n2, ok := tree.Node(uid)
		assert.True(t, ok)
		assert.Equal(t, n1, n2)

	})
	t.Run("should report when node does not exist", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		_, ok := tree.Node("invalid")
		assert.False(t, ok)
	})
}

func TestTreeNodes_MustNode(t *testing.T) {
	t.Run("can return a node", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		n1 := MyNode{"1", "Alpha"}
		uid := tree.MustAdd("", n1)
		n2 := tree.MustNode(uid)
		assert.Equal(t, n1, n2)
	})
	t.Run("should panic when node does not exist", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		assert.Panics(t, func() {
			tree.MustNode("invalid")

		})
	})
}

func TestTreeNodes_Path(t *testing.T) {
	t.Run("should return path for an existing node", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		uid1 := tree.MustAdd("", MyNode{"1", "Alpha"})
		uid2 := tree.MustAdd(uid1, MyNode{"2", "Bravo"})
		uid3 := tree.MustAdd(uid2, MyNode{"3", "Charlie"})
		p := tree.Path(uid3)
		assert.Equal(t, []widget.TreeNodeID{uid1, uid2}, p)
	})
	t.Run("should return empty array for root node", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		p := tree.Path("")
		assert.Equal(t, []widget.TreeNodeID{}, p)
	})
	t.Run("should return empty array for a top node", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		uid := tree.MustAdd("", MyNode{"1", "Alpha"})
		p := tree.Path(uid)
		assert.Equal(t, []widget.TreeNodeID{}, p)
	})
}

func TestTreeNodes_Parent(t *testing.T) {
	t.Run("can return parent of a top node", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		uid := tree.MustAdd("", MyNode{"1", "Alpha"})
		p, ok := tree.Parent(uid)
		assert.True(t, ok)
		assert.Equal(t, "", p)
	})

	t.Run("can return parent of a random node", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		uid1 := tree.MustAdd("", MyNode{"1", "Alpha"})
		uid2 := tree.MustAdd(uid1, MyNode{"2", "Bravo"})
		p, ok := tree.Parent(uid2)
		assert.True(t, ok)
		assert.Equal(t, uid1, p)
	})
	t.Run("can report when a parent does not exist", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		_, ok := tree.Parent("1")
		assert.False(t, ok)
	})
}

func TestTreeNodes_Size(t *testing.T) {
	t.Run("can return size of tree with nodes", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		tree.MustAdd("", MyNode{"1", "Alpha"})
		got := tree.Size()
		assert.Equal(t, 1, got)
	})
	t.Run("can return size of zero tree", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		got := tree.Size()
		assert.Equal(t, 0, got)
	})
}

func TestTreeNodes_Clear(t *testing.T) {
	t.Run("can clear tree with nodes", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		tree.MustAdd("", MyNode{"1", "Alpha"})
		tree.Clear()
		assert.Equal(t, 0, tree.Size())
	})
	t.Run("can clear empty tree", func(t *testing.T) {
		var tree iwidget.TreeNodes[MyNode]
		tree.Clear()
		assert.Equal(t, 0, tree.Size())
	})
	t.Run("panics with specific error when object is undefined", func(t *testing.T) {
		var tree *iwidget.TreeNodes[MyNode]
		assert.PanicsWithError(t, iwidget.ErrUndefined.Error(), func() {
			tree.Clear()
		})
	})
}

func TestTree_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	tree := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n MyNode, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Value)
		},
	)
	var nodes iwidget.TreeNodes[MyNode]
	uid := nodes.MustAdd(iwidget.RootUID, MyNode{"1", "Root"})
	nodes.Add(uid, MyNode{"2", "Alpha"})
	nodes.Add(uid, MyNode{"3", "Bravo"})
	tree.Set(nodes)
	tree.OpenAllBranches()
	w := test.NewWindow(tree)
	defer w.Close()
	w.Resize(fyne.NewSquareSize(500))

	test.AssertImageMatches(t, "tree/minimal.png", w.Canvas().Capture())
}
