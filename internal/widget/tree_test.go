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
	nodes.Add(nil, root, true)
	nodes.Add(root, &Node{"Alpha"}, false)
	nodes.Add(root, &Node{"Bravo"}, false)
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
	nodes.Add(nil, root, true)
	nodes.Add(root, &Node{"Alpha"}, false)
	nodes.Add(root, &Node{"Bravo"}, false)
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
	nodes.Add(nil, root, true)
	nodes.Add(root, &Node{"Alpha"}, false)
	nodes.Add(root, &Node{"Bravo"}, false)
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
	nodes.Add(nil, root, true)
	alpha := &Node{"Alpha"}
	nodes.Add(root, alpha, false)
	nodes.Add(root, &Node{"Bravo"}, false)
	tree.Set(nodes)

	tree.SelectNode(alpha)

	assert.Equal(t, alpha, selected)
}

func TestTreeData_Add(t *testing.T) {
	t.Run("can add a leaf node to root", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		n := &Node{"Alpha"}
		err := td.Add(nil, n, false)
		require.NoError(t, err)
		assert.True(t, td.Exists(n))
		assert.False(t, td.IsBranch(n))
	})
	t.Run("can add a branch node to root", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		n := &Node{"Alpha"}
		err := td.Add(nil, n, true)
		require.NoError(t, err)
		assert.True(t, td.Exists(n))
		assert.True(t, td.IsBranch(n))
	})
	t.Run("can add a node to another node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		a := &Node{"Alpha"}
		b := &Node{"Bravo"}
		err := td.Add(nil, a, true)
		require.NoError(t, err)
		err = td.Add(a, b, false)
		require.NoError(t, err)
		assert.True(t, td.Exists(b))
	})
	t.Run("should return error when parent does not exist", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		invalid := &Node{}
		err := td.Add(invalid, &Node{"Alpha"}, false)
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
	t.Run("should return error when trying to add a nil node", func(t *testing.T) {
		var td *iwidget.TreeData[Node]
		err := td.Add(nil, nil, true)
		assert.ErrorIs(t, err, iwidget.ErrInvalid)
	})
	t.Run("should return error when trying to add a node to a non-branch", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		a := &Node{"Alpha"}
		b := &Node{"Bravo"}
		err := td.Add(nil, a, false)
		require.NoError(t, err)
		err = td.Add(a, b, false)
		assert.ErrorIs(t, err, iwidget.ErrInvalid)
	})
}

func TestTreeData_Children(t *testing.T) {
	t.Run("can return children of a node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		top := &Node{"Top"}
		td.Add(nil, top, true)
		td.Add(top, &Node{"Alpha"}, false)
		td.Add(top, &Node{"Bravo"}, false)
		got := td.Children(top)
		want := []*Node{{"Alpha"}, {"Bravo"}}
		assert.Equal(t, want, got)
	})
	t.Run("can return children of root", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		top := &Node{"Top"}
		td.Add(nil, top, true)
		td.Add(top, &Node{"Alpha"}, false)
		td.Add(top, &Node{"Bravo"}, false)
		got := td.Children(nil)
		want := []*Node{top}
		assert.Equal(t, want, got)
	})
	t.Run("returns empty when a node has no children", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		top := &Node{"Top"}
		td.Add(nil, top, true)
		got := td.Children(top)
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
		branch1 := &Node{"Branch1"}
		td.Add(nil, branch1, true)
		td.Add(branch1, &Node{"Alpha"}, false)
		td.Add(branch1, &Node{"Bravo"}, false)
		branch2 := &Node{"Branch2"}
		td.Add(nil, branch2, true)
		td.Add(branch2, &Node{"Alpha2"}, false)
		td.Add(branch2, &Node{"Bravo2"}, false)
		got, ok := td.ChildrenCount(branch1)
		require.True(t, ok)
		assert.Equal(t, 2, got)
	})
	t.Run("can return count for a root node2", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		branch1 := &Node{"Branch1"}
		td.Add(nil, branch1, true)
		td.Add(branch1, &Node{"Alpha"}, false)
		branch2 := &Node{"Branch2"}
		td.Add(nil, branch2, true)
		td.Add(branch2, &Node{"Alpha2"}, false)
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
		td.Add(nil, alpha, true)
		n2 := &Node{"Bravo"}
		td.Add(nil, n2, true)
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
		td.Add(nil, n1, true)
		td.Add(n1, &Node{"Alpha"}, false)
		a := &Node{"Branch2"}
		td.Add(nil, a, true)
		b := &Node{"Bravo"}
		td.Add(a, b, false)
		c := &Node{"Charlie"}
		td.Add(nil, c, true)
		td.Print(nil, func(n *Node) string {
			return n.String()
		})
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
		td.Add(nil, &Node{"Alpha"}, true)
		invalid := &Node{}
		err := td.Delete(invalid)
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
}

func TestTreeData_IsEmpty(t *testing.T) {
	t.Run("can report non-empty td", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		td.Add(nil, &Node{"Root1"}, true)
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
		td.Add(nil, n1, true)
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
		td.Add(nil, alpha, true)
		bravo := &Node{"Bravo"}
		td.Add(alpha, bravo, false)
		p, ok := td.Parent(bravo)
		assert.True(t, ok)
		assert.Equal(t, alpha, p)
	})
	t.Run("the parent of a top node is the root node", func(t *testing.T) {
		var td iwidget.TreeData[Node]
		alpha := &Node{"Alpha"}
		td.Add(nil, alpha, true)
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
		td.Add(nil, a, true)
		b := &Node{"Bravo"}
		td.Add(a, b, true)
		c := &Node{"Charlie"}
		td.Add(b, c, false)
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
	td.Add(nil, root, true)
	alpha := &Node{"Alpha"}
	td.Add(root, alpha, false)
	bravo := &Node{"Bravo"}
	td.Add(root, bravo, false)
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
		td.Add(nil, &Node{"Alpha"}, true)
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
		td.Add(nil, &Node{"Alpha"}, true)
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
	td.Add(nil, top, true)

	a := &Node{"Alpha"}
	td.Add(top, a, true)

	c := &Node{"Charlie"}
	td.Add(a, c, false)

	b := &Node{"Bravo"}
	td.Add(top, b, false)

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
	td.Add(nil, top, true)

	a := &Node{"Alpha"}
	td.Add(top, a, true)

	c := &Node{"Charlie"}
	td.Add(a, c, false)

	b := &Node{"Bravo"}
	td.Add(top, b, false)

	got1 := td.AllPaths(nil, func(n *Node) string {
		return n.String()
	})
	want1 := [][]string{
		{"Top", "Bravo"},
		{"Top", "Alpha", "Charlie"},
	}
	assert.ElementsMatch(t, want1, got1)

	got2 := td.AllPaths(a, func(n *Node) string {
		return n.String()
	})
	want2 := [][]string{
		{"Alpha", "Charlie"},
	}
	assert.ElementsMatch(t, want2, got2)
}

func TestTreeData_SortChildren(t *testing.T) {
	var td iwidget.TreeData[Node]
	top := &Node{"Top"}
	td.Add(nil, top, true)

	b := &Node{"Bravo"}
	td.Add(top, b, false)

	a := &Node{"Alpha"}
	td.Add(top, a, false)

	td.SortChildrenFunc(top, func(a, b *Node) int {
		return strings.Compare(a.Text, b.Text)
	})
	got := td.Children(top)
	want := []*Node{a, b}
	assert.ElementsMatch(t, want, got)
}

func TestTreeData_DeleteChildren(t *testing.T) {
	var td iwidget.TreeData[Node]
	top := &Node{"Top"}
	td.Add(nil, top, true)

	b := &Node{"Bravo"}
	td.Add(top, b, false)

	a := &Node{"Alpha"}
	td.Add(top, a, false)

	td.DeleteChildrenFunc(top, func(n *Node) bool {
		return n.Text == "Alpha"
	})
	got := td.Children(top)
	want := []*Node{b}
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
	td.Add(nil, top, true) // adds to root
	td.Add(top, &Node{"Alpha"}, false)
	td.Add(top, &Node{"Bravo"}, false)

	// Update tree
	tree.Set(td)

	w.SetContent(tree)
	w.Resize(fyne.NewSize(600, 400))
	w.ShowAndRun()
}
