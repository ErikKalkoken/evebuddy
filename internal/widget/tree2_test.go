package widget_test

import (
	"fmt"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type MyNode2 struct {
	Text string
}

func (x MyNode2) String() string {
	return x.Text
}

func TestTree2_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	tree := iwidget.NewTree2(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n *MyNode2, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Text)
		},
	)
	var nodes iwidget.TreeData2[MyNode2]
	root := &MyNode2{"Root"}
	nodes.Add(nil, root, true)
	nodes.Add(root, &MyNode2{"Alpha"}, false)
	nodes.Add(root, &MyNode2{"Bravo"}, false)
	tree.Set(nodes)
	tree.OpenAllBranches()
	w := test.NewWindow(tree)
	defer w.Close()
	w.Resize(fyne.NewSquareSize(500))

	test.AssertImageMatches(t, "tree/minimal.png", w.Canvas().Capture())
}

func TestTree2_CanReturnNodes(t *testing.T) {
	test.NewTempApp(t)

	tree := iwidget.NewTree2(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n *MyNode2, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Text)
		},
	)
	var nodes iwidget.TreeData2[MyNode2]
	root := &MyNode2{"Root"}
	nodes.Add(nil, root, true)
	nodes.Add(root, &MyNode2{"Alpha"}, true)
	nodes.Add(root, &MyNode2{"Bravo"}, true)
	tree.Set(nodes)

	assert.IsType(t, iwidget.TreeData2[MyNode2]{}, tree.Data())
}

func TestTree2_CanClear(t *testing.T) {
	test.NewTempApp(t)

	tree := iwidget.NewTree2(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n *MyNode2, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Text)
		},
	)
	var nodes iwidget.TreeData2[MyNode2]
	root := &MyNode2{"Root"}
	nodes.Add(nil, root, true)
	nodes.Add(root, &MyNode2{"Alpha"}, true)
	nodes.Add(root, &MyNode2{"Bravo"}, true)
	tree.Set(nodes)

	tree.Clear()

	got := tree.Data()
	assert.True(t, got.IsEmpty())
}

func TestTree2_OnSelectedNode(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	tree := iwidget.NewTree2(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n *MyNode2, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Text)
		},
	)
	var selected *MyNode2
	tree.OnSelectedNode = func(n *MyNode2) {
		selected = n
	}
	var nodes iwidget.TreeData2[MyNode2]
	root := &MyNode2{"Root"}
	nodes.Add(nil, root, true)
	alpha := &MyNode2{"Alpha"}
	nodes.Add(root, alpha, true)
	nodes.Add(root, &MyNode2{"Bravo"}, true)
	tree.Set(nodes)

	tree.SelectNode(alpha)

	assert.Equal(t, alpha, selected)
}

func TestTreeData2_Add(t *testing.T) {
	t.Run("can add a node", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		n := &MyNode2{"Alpha"}
		err := tree.Add(nil, n, true)
		require.NoError(t, err)
		assert.True(t, tree.Exists(n))
	})
	t.Run("should return error when parent does not exist", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		invalid := &MyNode2{}
		err := tree.Add(invalid, &MyNode2{"Alpha"}, true)
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
	t.Run("can add a node to zero value", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		n := &MyNode2{"Alpha"}
		err := tree.Add(nil, n, true)
		require.NoError(t, err)
		assert.True(t, tree.Exists(n))
	})
	t.Run("should return error when trying to add a nil pointer", func(t *testing.T) {
		var tree *iwidget.TreeData2[MyNode2]
		err := tree.Add(nil, nil, true)
		assert.ErrorIs(t, err, iwidget.ErrInvalid)
	})
}

func TestTreeData2_Children(t *testing.T) {
	t.Run("can return children of a node", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		top := &MyNode2{"Top"}
		tree.Add(nil, top, true)
		tree.Add(top, &MyNode2{"Alpha"}, false)
		tree.Add(top, &MyNode2{"Bravo"}, false)
		got := tree.Children(top)
		want := []*MyNode2{{"Alpha"}, {"Bravo"}}
		assert.Equal(t, want, got)
	})
	t.Run("can return children of root", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		top := &MyNode2{"Top"}
		tree.Add(nil, top, true)
		tree.Add(top, &MyNode2{"Alpha"}, true)
		tree.Add(top, &MyNode2{"Bravo"}, true)
		got := tree.Children(nil)
		want := []*MyNode2{top}
		assert.Equal(t, want, got)
	})
	t.Run("returns empty when a node has no children", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		root := &MyNode2{"Top"}
		tree.Add(nil, root, true)
		got := tree.Children(root)
		assert.Len(t, got, 0)
	})
	t.Run("the root always exists", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		got := tree.Children(nil)
		assert.Len(t, got, 0)
	})
	t.Run("should return empty slice when node was not found", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		invalid := &MyNode2{}
		got := tree.Children(invalid)
		assert.Len(t, got, 0)
	})
}

func TestTreeData2_ChildrenCount(t *testing.T) {
	t.Run("can return count for a node2", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		branch1 := &MyNode2{"Root1"}
		tree.Add(nil, branch1, true)
		tree.Add(branch1, &MyNode2{"Alpha"}, true)
		tree.Add(branch1, &MyNode2{"Bravo"}, true)
		branch2 := &MyNode2{"Root2"}
		tree.Add(nil, branch2, true)
		tree.Add(branch2, &MyNode2{"Alpha2"}, true)
		tree.Add(branch2, &MyNode2{"Bravo2"}, true)
		got, ok := tree.ChildrenCount(branch1)
		require.True(t, ok)
		assert.Equal(t, 2, got)
	})
	t.Run("can return count for a root node2", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		branch1 := &MyNode2{"Root1"}
		tree.Add(nil, branch1, true)
		tree.Add(branch1, &MyNode2{"Alpha"}, true)
		branch2 := &MyNode2{"Root2"}
		tree.Add(nil, branch2, true)
		tree.Add(branch2, &MyNode2{"Alpha2"}, true)
		got, ok := tree.ChildrenCount(nil)
		require.True(t, ok)
		assert.Equal(t, 2, got)
	})
	t.Run("can return the count for of an empty tree", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		got, ok := tree.ChildrenCount(nil)
		require.True(t, ok)
		assert.Equal(t, 0, got)
	})
	t.Run("should return error when node was not found", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		invalid := &MyNode2{}
		_, ok := tree.ChildrenCount(invalid)
		assert.False(t, ok)
	})
}

func TestTreeData2_Delete(t *testing.T) {
	t.Run("can remove a node from simple tree", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		alpha := &MyNode2{"Alpha"}
		tree.Add(nil, alpha, true)
		n2 := &MyNode2{"Bravo"}
		tree.Add(nil, n2, true)
		err := tree.Delete(n2)
		require.NoError(t, err)
		assert.ElementsMatch(t, slices.Collect(tree.All()), []*MyNode2{alpha})
	})
	t.Run("can remove node from complex tree", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		n1 := &MyNode2{"Branch1"}
		tree.Add(nil, n1, true)
		tree.Add(n1, &MyNode2{"Alpha"}, true)
		a := &MyNode2{"Branch2"}
		tree.Add(nil, a, true)
		b := &MyNode2{"Bravo"}
		tree.Add(a, b, true)
		c := &MyNode2{"Charlie"}
		tree.Add(nil, c, true)
		tree.Print(nil)
		err := tree.Delete(n1)
		require.NoError(t, err)
		assert.ElementsMatch(t, slices.Collect(tree.All()), []*MyNode2{a, b, c})
		// t.Fail()
	})
	t.Run("can not remove the root node", func(t *testing.T) {
		var td iwidget.TreeData2[MyNode2]
		err := td.Delete(nil)
		assert.ErrorIs(t, err, iwidget.ErrInvalid)
	})
	t.Run("can not remove a node from an empty tree", func(t *testing.T) {
		var td iwidget.TreeData2[MyNode2]
		err := td.Delete(&MyNode2{})
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
	t.Run("return error when trying to remove non-existing node", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		tree.Add(nil, &MyNode2{"Alpha"}, true)
		invalid := &MyNode2{}
		err := tree.Delete(invalid)
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
}

func TestTreeData2_IsEmpty(t *testing.T) {
	t.Run("can report non-empty tree", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		tree.Add(nil, &MyNode2{"Root1"}, true)
		assert.Equal(t, false, tree.IsEmpty())
	})
	t.Run("can report empty tree", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		assert.Equal(t, true, tree.IsEmpty())
	})
}

func TestTreeData2_Node(t *testing.T) {
	t.Run("should return node2 when it exists", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		n1 := &MyNode2{"Alpha"}
		tree.Add(nil, n1, true)
		uid, ok := tree.UID(n1)
		require.True(t, ok)
		n2, ok := tree.Node(uid)
		require.True(t, ok)
		assert.Equal(t, n1, n2)

	})
	t.Run("should report when node does not exist", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		_, ok := tree.Node("invalid")
		assert.False(t, ok)
	})
	t.Run("the root node does exist", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		_, ok := tree.Node(iwidget.TreeRootID)
		assert.True(t, ok)
	})
}

func TestTreeData2_Parent(t *testing.T) {
	t.Run("can return parent of a node", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		alpha := &MyNode2{"Alpha"}
		tree.Add(nil, alpha, true)
		bravo := &MyNode2{"Bravo"}
		tree.Add(alpha, bravo, true)
		p, ok := tree.Parent(bravo)
		assert.True(t, ok)
		assert.Equal(t, alpha, p)
	})
	t.Run("the parent of a top node is the root node", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		alpha := &MyNode2{"Alpha"}
		tree.Add(nil, alpha, true)
		p, ok := tree.Parent(alpha)
		assert.True(t, ok)
		assert.Nil(t, p)
	})
	t.Run("can report when a parent does not exist", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		invalid := &MyNode2{}
		_, ok := tree.Parent(invalid)
		assert.False(t, ok)
	})
	t.Run("the root node has no parents", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		_, ok := tree.Parent(nil)
		assert.False(t, ok)
	})
}

func TestTreeData2_Path(t *testing.T) {
	t.Run("should return path for an existing node2", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		a := &MyNode2{"Alpha"}
		tree.Add(nil, a, true)
		b := &MyNode2{"Bravo"}
		tree.Add(a, b, true)
		c := &MyNode2{"Charlie"}
		tree.Add(b, c, true)
		p := tree.Path(c)
		assert.Equal(t, []*MyNode2{a, b}, p)
	})
	t.Run("should return empty array for root node", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		p := tree.Path(nil)
		assert.Equal(t, []*MyNode2{}, p)
	})
}

func TestTreeData2_Values(t *testing.T) {
	var nodes iwidget.TreeData2[MyNode2]
	root := &MyNode2{"Root"}
	nodes.Add(nil, root, true)
	alpha := &MyNode2{"Alpha"}
	nodes.Add(root, alpha, true)
	bravo := &MyNode2{"Bravo"}
	nodes.Add(root, bravo, true)
	got := slices.Collect(nodes.All())
	want := []*MyNode2{root, alpha, bravo}
	assert.ElementsMatch(t, want, got)

}

func TestTreeData2_Clear(t *testing.T) {
	t.Run("can clear tree with nodes", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		tree.Add(nil, &MyNode2{"Alpha"}, true)
		tree.Clear()
		assert.True(t, tree.IsEmpty())
	})
	t.Run("can clear empty tree", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		tree.Clear()
		assert.True(t, tree.IsEmpty())
	})
	t.Run("can clear nill tree", func(t *testing.T) {
		var tree *iwidget.TreeData2[MyNode2]
		tree.Clear()
	})
}

func TestTreeData2_Size(t *testing.T) {
	t.Run("can return size of tree with nodes", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		tree.Add(nil, &MyNode2{"Alpha"}, true)
		got := tree.Size()
		assert.Equal(t, 1, got)
	})
	t.Run("can return size of zero tree", func(t *testing.T) {
		var tree iwidget.TreeData2[MyNode2]
		got := tree.Size()
		assert.Equal(t, 0, got)
	})
}

func TestTreeData2_String(t *testing.T) {
	var tree iwidget.TreeData2[MyNode2]
	alpha := &MyNode2{"Alpha"}
	tree.Add(nil, alpha, true)
	tree.Add(alpha, &MyNode2{"Bravo"}, true)
	s := fmt.Sprint(tree)
	assert.Equal(t, "{nodes map[1:Alpha 2:Bravo], children: map[:[1] 1:[2]]}", s)
}
