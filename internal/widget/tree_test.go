package widget_test

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type Node struct {
	Text string
}

func (n Node) String() string {
	return n.Text
}

func TestTree_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	tree := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n *Node, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Text)
		},
	)
	var nodes iwidget.TreeData[Node]
	root := &Node{"Root"}
	nodes.Add(nil, root)
	nodes.Add(root, &Node{"Alpha"})
	nodes.Add(root, &Node{"Bravo"})
	tree.Set(nodes)
	tree.OpenAllBranches()
	w := test.NewWindow(tree)
	defer w.Close()
	w.Resize(fyne.NewSquareSize(500))

	test.AssertImageMatches(t, "tree/minimal.png", w.Canvas().Capture())
}

func TestTree_CanReturnNodes(t *testing.T) {
	test.NewTempApp(t)

	tree := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n *Node, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Text)
		},
	)
	var nodes iwidget.TreeData[Node]
	root := &Node{"Root"}
	nodes.Add(nil, root)
	nodes.Add(root, &Node{"Alpha"})
	nodes.Add(root, &Node{"Bravo"})
	tree.Set(nodes)

	assert.IsType(t, iwidget.TreeData[Node]{}, tree.Data())
}

func TestTree_CanClear(t *testing.T) {
	test.NewTempApp(t)

	tree := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n *Node, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Text)
		},
	)
	var nodes iwidget.TreeData[Node]
	root := &Node{"Root"}
	nodes.Add(nil, root)
	nodes.Add(root, &Node{"Alpha"})
	nodes.Add(root, &Node{"Bravo"})
	tree.Set(nodes)

	tree.Clear()

	got := tree.Data()
	assert.True(t, got.IsEmpty())
}

func TestTree_OnSelectedNode(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	tree := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n *Node, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Text)
		},
	)
	var selected *Node
	tree.OnSelectedNode = func(n *Node) {
		selected = n
	}
	var nodes iwidget.TreeData[Node]
	root := &Node{"Root"}
	nodes.Add(nil, root)
	alpha := &Node{"Alpha"}
	nodes.Add(root, alpha)
	nodes.Add(root, &Node{"Bravo"})
	tree.Set(nodes)

	tree.SelectNode(alpha)

	assert.Equal(t, alpha, selected)
}

func TestTreeData_Add(t *testing.T) {
	t.Run("can add a node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		n := &Node{"Alpha"}
		err := td.Add(nil, n)
		require.NoError(t, err)
		assert.True(t, td.Exists(n))
	})
	t.Run("should return error when parent does not exist", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		invalid := &Node{}
		err := td.Add(invalid, &Node{"Alpha"})
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
	t.Run("can add a node to zero value", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		n := &Node{"Alpha"}
		err := td.Add(nil, n)
		require.NoError(t, err)
		assert.True(t, td.Exists(n))
	})
	t.Run("should return error when trying to add a nil pointer", func(t *testing.T) {
		var td *iwidget.TreeData[Node]
		err := td.Add(nil, nil)
		assert.ErrorIs(t, err, iwidget.ErrInvalid)
	})
}

func TestTreeData_Children(t *testing.T) {
	t.Run("can return children of a node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		top := &Node{"Top"}
		td.Add(nil, top)
		td.Add(top, &Node{"Alpha"})
		td.Add(top, &Node{"Bravo"})
		got := td.Children(top)
		want := []*Node{{"Alpha"}, {"Bravo"}}
		assert.Equal(t, want, got)
	})
	t.Run("can return children of root", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		top := &Node{"Top"}
		td.Add(nil, top)
		td.Add(top, &Node{"Alpha"})
		td.Add(top, &Node{"Bravo"})
		got := td.Children(nil)
		want := []*Node{top}
		assert.Equal(t, want, got)
	})
	t.Run("returns empty when a node has no children", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		root := &Node{"Top"}
		td.Add(nil, root)
		got := td.Children(root)
		assert.Len(t, got, 0)
	})
	t.Run("the root always exists", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		got := td.Children(nil)
		assert.Len(t, got, 0)
	})
	t.Run("should return empty slice when node was not found", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		invalid := &Node{}
		got := td.Children(invalid)
		assert.Len(t, got, 0)
	})
}

func TestTreeData_ChildrenCount(t *testing.T) {
	t.Run("can return count for a node2", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		branch1 := &Node{"Root1"}
		td.Add(nil, branch1)
		td.Add(branch1, &Node{"Alpha"})
		td.Add(branch1, &Node{"Bravo"})
		branch2 := &Node{"Root2"}
		td.Add(nil, branch2)
		td.Add(branch2, &Node{"Alpha2"})
		td.Add(branch2, &Node{"Bravo2"})
		got, ok := td.ChildrenCount(branch1)
		require.True(t, ok)
		assert.Equal(t, 2, got)
	})
	t.Run("can return count for a root node2", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		branch1 := &Node{"Root1"}
		td.Add(nil, branch1)
		td.Add(branch1, &Node{"Alpha"})
		branch2 := &Node{"Root2"}
		td.Add(nil, branch2)
		td.Add(branch2, &Node{"Alpha2"})
		got, ok := td.ChildrenCount(nil)
		require.True(t, ok)
		assert.Equal(t, 2, got)
	})
	t.Run("can return the count for of an empty td", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		got, ok := td.ChildrenCount(nil)
		require.True(t, ok)
		assert.Equal(t, 0, got)
	})
	t.Run("should return error when node was not found", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		invalid := &Node{}
		_, ok := td.ChildrenCount(invalid)
		assert.False(t, ok)
	})
}

func TestTreeData_Delete(t *testing.T) {
	t.Run("can remove a node from a simple td", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		alpha := &Node{"Alpha"}
		td.Add(nil, alpha)
		n2 := &Node{"Bravo"}
		td.Add(nil, n2)
		err := td.Delete(n2)
		require.NoError(t, err)
		want := make([]*Node, 0)
		td.Walk(nil, func(n *Node) bool {
			want = append(want, n)
			return true
		})
		assert.ElementsMatch(t, want, []*Node{alpha})
	})
	t.Run("can remove node from a complex td", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		n1 := &Node{"Branch1"}
		td.Add(nil, n1)
		td.Add(n1, &Node{"Alpha"})
		a := &Node{"Branch2"}
		td.Add(nil, a)
		b := &Node{"Bravo"}
		td.Add(a, b)
		c := &Node{"Charlie"}
		td.Add(nil, c)
		td.Print(nil)
		err := td.Delete(n1)
		require.NoError(t, err)
		want := make([]*Node, 0)
		td.Walk(nil, func(n *Node) bool {
			want = append(want, n)
			return true
		})
		assert.ElementsMatch(t, want, []*Node{a, b, c})
		// t.Fail()
	})
	t.Run("can not remove the root node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		err := td.Delete(nil)
		assert.ErrorIs(t, err, iwidget.ErrInvalid)
	})
	t.Run("can not remove a node from an empty td", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		err := td.Delete(&Node{})
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
	t.Run("return error when trying to remove non-existing node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		td.Add(nil, &Node{"Alpha"})
		invalid := &Node{}
		err := td.Delete(invalid)
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
}

func TestTreeData_IsEmpty(t *testing.T) {
	t.Run("can report non-empty td", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		td.Add(nil, &Node{"Root1"})
		assert.Equal(t, false, td.IsEmpty())
	})
	t.Run("can report empty td", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		assert.Equal(t, true, td.IsEmpty())
	})
}

func TestTreeData_Node(t *testing.T) {
	t.Run("should return node2 when it exists", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		n1 := &Node{"Alpha"}
		td.Add(nil, n1)
		uid, ok := td.UID(n1)
		require.True(t, ok)
		n2, ok := td.Node(uid)
		require.True(t, ok)
		assert.Equal(t, n1, n2)

	})
	t.Run("should report when node does not exist", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		_, ok := td.Node("invalid")
		assert.False(t, ok)
	})
}

func TestTreeData_Parent(t *testing.T) {
	t.Run("can return parent of a node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		alpha := &Node{"Alpha"}
		td.Add(nil, alpha)
		bravo := &Node{"Bravo"}
		td.Add(alpha, bravo)
		p, ok := td.Parent(bravo)
		assert.True(t, ok)
		assert.Equal(t, alpha, p)
	})
	t.Run("the parent of a top node is the root node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		alpha := &Node{"Alpha"}
		td.Add(nil, alpha)
		p, ok := td.Parent(alpha)
		assert.True(t, ok)
		assert.Nil(t, p)
	})
	t.Run("can report when a parent does not exist", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		invalid := &Node{}
		_, ok := td.Parent(invalid)
		assert.False(t, ok)
	})
	t.Run("the root node has no parents", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		_, ok := td.Parent(nil)
		assert.False(t, ok)
	})
}

func TestTreeData_Path(t *testing.T) {
	t.Run("should return path for an existing node2", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		a := &Node{"Alpha"}
		td.Add(nil, a)
		b := &Node{"Bravo"}
		td.Add(a, b)
		c := &Node{"Charlie"}
		td.Add(b, c)
		p := td.Path(nil, c)
		assert.Equal(t, []*Node{a, b, c}, p)
	})
	t.Run("should return empty slice for root node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		p := td.Path(nil, nil)
		assert.Empty(t, p)
	})
}

func TestTreeData_Values(t *testing.T) {
	var td iwidget.TreeData[Node]
	root := &Node{"Root"}
	td.Add(nil, root)
	alpha := &Node{"Alpha"}
	td.Add(root, alpha)
	bravo := &Node{"Bravo"}
	td.Add(root, bravo)
	got := make([]*Node, 0)
	td.Walk(nil, func(n *Node) bool {
		got = append(got, n)
		return true
	})
	want := []*Node{root, alpha, bravo}
	assert.ElementsMatch(t, want, got)

}

func TestTreeData_Clear(t *testing.T) {
	t.Run("can clear td with nodes", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		td.Add(nil, &Node{"Alpha"})
		td.Clear()
		assert.True(t, td.IsEmpty())
	})
	t.Run("can clear empty td", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		td.Clear()
		assert.True(t, td.IsEmpty())
	})
	t.Run("can clear nill td", func(t *testing.T) {
		var td *iwidget.TreeData[Node]
		td.Clear()
	})
}

func TestTreeData_Size(t *testing.T) {
	t.Run("can return size of td with nodes", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		td.Add(nil, &Node{"Alpha"})
		got := td.Size()
		assert.Equal(t, 1, got)
	})
	t.Run("can return size of zero td", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		got := td.Size()
		assert.Equal(t, 0, got)
	})
}

func TestTreeData_Walk(t *testing.T) {
	var td iwidget.TreeData[Node]
	top := &Node{"Top"}
	td.Add(nil, top)

	a := &Node{"Alpha"}
	td.Add(top, a)

	c := &Node{"Charlie"}
	td.Add(a, c)

	b := &Node{"Bravo"}
	td.Add(top, b)

	got := make([]*Node, 0)
	td.Walk(nil, func(n *Node) bool {
		got = append(got, n)
		return true
	})
	want := []*Node{top, a, c, b}
	assert.Equal(t, want, got)
}

func TestTreeData_AllPaths(t *testing.T) {
	var td iwidget.TreeData[Node]
	top := &Node{"Top"}
	td.Add(nil, top)

	a := &Node{"Alpha"}
	td.Add(top, a)

	c := &Node{"Charlie"}
	td.Add(a, c)

	b := &Node{"Bravo"}
	td.Add(top, b)

	got1 := td.AllPaths(nil)
	want1 := [][]string{
		{"Top", "Bravo"},
		{"Top", "Alpha", "Charlie"},
	}
	assert.ElementsMatch(t, want1, got1)

	got2 := td.AllPaths(a)
	want2 := [][]string{
		{"Alpha", "Charlie"},
	}
	assert.ElementsMatch(t, want2, got2)
}

func TestTreeData_SortChilren(t *testing.T) {
	var td iwidget.TreeData[Node]
	top := &Node{"Top"}
	td.Add(nil, top)

	b := &Node{"Bravo"}
	td.Add(top, b)

	a := &Node{"Alpha"}
	td.Add(top, a)

	td.SortChildren(top, func(a, b *Node) int {
		return strings.Compare(a.Text, b.Text)
	})
	got := td.Children(top)
	want := []*Node{a, b}
	assert.ElementsMatch(t, want, got)

}

func ExampleTree() {
	a := app.New()
	w := a.NewWindow("Tree Example")

	// Create tree widget
	tree := iwidget.NewTree(
		func(_ bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n *Node, _ bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Text)
		},
	)

	// Create tree data
	var td iwidget.TreeData[Node]
	top := &Node{"Top"}
	td.Add(nil, top) // adds to root
	td.Add(top, &Node{"Alpha"})
	td.Add(top, &Node{"Bravo"})

	// Update tree
	tree.Set(td)

	w.SetContent(tree)
	w.Resize(fyne.NewSize(600, 400))
	w.ShowAndRun()
}
