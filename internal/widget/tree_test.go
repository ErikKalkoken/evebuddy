package widget_test

import (
	"fmt"
	"slices"
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
	var nodes iwidget.TreeData[MyNode]
	root := nodes.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root"})
	nodes.Add(root, MyNode{"2", "Alpha"})
	nodes.Add(root, MyNode{"3", "Bravo"})
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
		func(n MyNode, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Value)
		},
	)
	var nodes iwidget.TreeData[MyNode]
	root := nodes.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root"})
	nodes.Add(root, MyNode{"2", "Alpha"})
	nodes.Add(root, MyNode{"3", "Bravo"})
	tree.Set(nodes)

	got := tree.Data()
	assert.True(t, nodes.Equal(got), "got %q, wanted %q", got, nodes)
}

func TestTree_CanClear(t *testing.T) {
	test.NewTempApp(t)

	tree := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(n MyNode, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Value)
		},
	)
	var nodes iwidget.TreeData[MyNode]
	root := nodes.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root"})
	nodes.Add(root, MyNode{"2", "Alpha"})
	nodes.Add(root, MyNode{"3", "Bravo"})
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
		func(n MyNode, isBranch bool, co fyne.CanvasObject) {
			co.(*widget.Label).SetText(n.Value)
		},
	)
	var selected MyNode
	tree.OnSelectedNode = func(n MyNode) {
		selected = n
	}
	var nodes iwidget.TreeData[MyNode]
	root := nodes.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root"})
	alpha := MyNode{"2", "Alpha"}
	nodes.Add(root, alpha)
	nodes.Add(root, MyNode{"3", "Bravo"})
	tree.Set(nodes)

	tree.Select(alpha.UID())

	assert.Equal(t, alpha.UID(), selected.UID())
}

func TestTreeData_CanCreateFullTree(t *testing.T) {
	var tree iwidget.TreeData[MyNode]
	b1 := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
	n11 := tree.MustAdd(b1, MyNode{"11", "one"})
	n12 := tree.MustAdd(b1, MyNode{"12", "two"})
	b2 := tree.MustAdd(iwidget.TreeRootID, MyNode{"2", "Bravo"})
	assert.Equal(t, 5, tree.Size())
	assert.Equal(t, []string{n11, n12}, tree.ChildUIDs(b1))
	assert.Len(t, tree.ChildUIDs(b2), 0)
	nodes := []struct {
		uid      string
		value    string
		isBranch bool
	}{
		{b1, "Alpha", true},
		{n11, "one", false},
		{n12, "two", false},
		{b2, "Bravo", false},
	}
	for _, x := range nodes {
		b := node(tree, x.uid)
		assert.Equal(t, MyNode{x.uid, x.value}, b)
		assert.Equal(t, x.isBranch, tree.IsBranch(x.uid))
	}
}

func TestTreeData_Add(t *testing.T) {
	t.Run("can add a node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		n := MyNode{"1", "Alpha"}
		uid, err := tree.Add(iwidget.TreeRootID, n)
		if assert.NoError(t, err) {
			assert.Equal(t, n, node(tree, uid))
		}
	})
	t.Run("should return error when node parent UID does not exist", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		_, err := tree.Add("invalid", MyNode{"1", "Alpha"})
		assert.Error(t, err)
	})
	t.Run("can add a node to zero value", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		n := MyNode{"1", "Alpha"}
		uid, err := tree.Add(iwidget.TreeRootID, n)
		if assert.NoError(t, err) {
			assert.Equal(t, n, node(tree, uid))
		}
	})
	t.Run("should return error when trying to add to a nil pointer", func(t *testing.T) {
		var tree *iwidget.TreeData[MyNode]
		_, err := tree.Add(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		assert.ErrorIs(t, err, iwidget.ErrInvalid)
	})
}

func TestTreeData_Children(t *testing.T) {
	t.Run("can return children of a node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		branch := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root1"})
		tree.Add(branch, MyNode{"2", "Alpha"})
		tree.Add(branch, MyNode{"3", "Bravo"})
		got, err := tree.Children(branch)
		if assert.NoError(t, err) {
			want := []MyNode{{"2", "Alpha"}, {"3", "Bravo"}}
			assert.Equal(t, want, got)
		}
	})
	t.Run("can return children of root", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		branch := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Branch"})
		tree.Add(branch, MyNode{"2", "Alpha"})
		tree.Add(branch, MyNode{"3", "Bravo"})
		got, err := tree.Children(iwidget.TreeRootID)
		if assert.NoError(t, err) {
			want := []MyNode{{"1", "Branch"}}
			assert.Equal(t, want, got)
		}
	})
	t.Run("returns empty if a node has no children", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		uid := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root1"})
		got, err := tree.Children(uid)
		if assert.NoError(t, err) {
			assert.Len(t, got, 0)
		}
	})
	t.Run("the root ID always exists", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		got, err := tree.Children(iwidget.TreeRootID)
		if assert.NoError(t, err) {
			assert.Len(t, got, 0)
		}
	})
	t.Run("should return error when node not found", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		_, err := tree.Children("abc")
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
}

func TestTreeData_ChildrenCount(t *testing.T) {
	t.Run("can return count for a node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		branch1 := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root1"})
		tree.Add(branch1, MyNode{"2", "Alpha"})
		tree.Add(branch1, MyNode{"3", "Bravo"})
		branch2 := tree.MustAdd(iwidget.TreeRootID, MyNode{"4", "Root2"})
		tree.Add(branch2, MyNode{"5", "Alpha2"})
		tree.Add(branch2, MyNode{"6", "Bravo2"})
		got, err := tree.ChildrenCount(branch1)
		if assert.NoError(t, err) {
			assert.Equal(t, 2, got)
		}
	})
	t.Run("can return count for a root node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		branch1 := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root1"})
		tree.Add(branch1, MyNode{"2", "Alpha"})
		branch2 := tree.MustAdd(iwidget.TreeRootID, MyNode{"4", "Root2"})
		tree.Add(branch2, MyNode{"5", "Alpha2"})
		got, err := tree.ChildrenCount(iwidget.TreeRootID)
		if assert.NoError(t, err) {
			assert.Equal(t, 2, got)
		}
	})
	t.Run("can return the count for of an empty tree", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		got, err := tree.ChildrenCount(iwidget.TreeRootID)
		if assert.NoError(t, err) {
			assert.Equal(t, 0, got)
		}
	})
	t.Run("should return error when node not found", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		_, err := tree.ChildrenCount("abc")
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
}

func TestTreeData_ChildUIDs(t *testing.T) {
	t.Run("can return child UIDs of existing node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		sub1 := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		c1 := tree.MustAdd(sub1, MyNode{"2", "Bravo"})
		c2 := tree.MustAdd(sub1, MyNode{"3", "Charlie"})
		sub2 := tree.MustAdd(iwidget.TreeRootID, MyNode{"4", "Delta"})
		tree.MustAdd(sub2, MyNode{"5", "Echo"})
		got := tree.ChildUIDs(sub1)
		want := []widget.TreeNodeID{c1, c2}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("should return empty when node does not exist", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		got := tree.ChildUIDs("xy")
		assert.Empty(t, got)
	})
	t.Run("should return empty when node has not children", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		n1 := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		got := tree.ChildUIDs(n1)
		assert.Empty(t, got)
	})
	t.Run("should return empty when tree is zero", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		got := tree.ChildUIDs("xx")
		assert.Empty(t, got)
	})
	t.Run("should return empty when tree is zero for root node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		got := tree.ChildUIDs(iwidget.TreeRootID)
		assert.Empty(t, got)
	})
}

func TestTreeData_Delete(t *testing.T) {
	t.Run("can remove a node from simple tree", func(t *testing.T) {
		var t1, t2 iwidget.TreeData[MyNode]
		n1 := t1.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		t1.MustAdd(iwidget.TreeRootID, MyNode{"2", "Bravo"})
		err := t1.Delete(n1)
		if assert.NoError(t, err) {
			t2.MustAdd(iwidget.TreeRootID, MyNode{"2", "Bravo"})
			assert.True(t, t1.Equal(t2), "got %q, wanted %q", t1, t2)
		}
	})
	t.Run("can remove node from complex tree", func(t *testing.T) {
		var t1 iwidget.TreeData[MyNode]
		n1 := t1.MustAdd(iwidget.TreeRootID, MyNode{"1", "Branch1"})
		t1.Add(n1, MyNode{"11", "Alpha"})
		n2 := t1.MustAdd(iwidget.TreeRootID, MyNode{"2", "Branch2"})
		t1.Add(n2, MyNode{"21", "Bravo"})
		t1.MustAdd(iwidget.TreeRootID, MyNode{"3", "Charlie"})
		err := t1.Delete(n1)
		if assert.NoError(t, err) {
			var t2 iwidget.TreeData[MyNode]
			n3 := t2.MustAdd(iwidget.TreeRootID, MyNode{"2", "Branch2"})
			t2.Add(n3, MyNode{"21", "Bravo"})
			t2.MustAdd(iwidget.TreeRootID, MyNode{"3", "Charlie"})
			assert.True(t, t1.Equal(t2), "got %q, wanted %q", t1, t2)
		}
	})
	t.Run("can not remove the root node", func(t *testing.T) {
		var td iwidget.TreeData[MyNode]
		err := td.Delete(iwidget.TreeRootID)
		assert.ErrorIs(t, err, iwidget.ErrInvalid)
	})
	t.Run("can not remove node from empty tree", func(t *testing.T) {
		var td iwidget.TreeData[MyNode]
		err := td.Delete("xxx")
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
	t.Run("return error when trying to remove non-existing node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		tree.Add(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		err := tree.Delete("xxx")
		assert.ErrorIs(t, err, iwidget.ErrNotFound)
	})
}

func TestTreeData_IsEmpty(t *testing.T) {
	t.Run("can report non-empty tree", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root1"})
		assert.Equal(t, false, tree.IsEmpty())
	})
	t.Run("can report empty tree", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		assert.Equal(t, true, tree.IsEmpty())
	})
}

func TestTreeData_MustAdd(t *testing.T) {
	t.Run("can add a node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		n := MyNode{"1", "Alpha"}
		uid := tree.MustAdd(iwidget.TreeRootID, n)
		assert.Equal(t, n, node(tree, uid))
	})
	t.Run("should return error when node UID already exists", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		_, err := tree.Add(iwidget.TreeRootID, MyNode{"1", "Bravo"})
		assert.Error(t, err)
	})
	t.Run("should return error when node UID is root", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		_, err := tree.Add(iwidget.TreeRootID, MyNode{iwidget.TreeRootID, "Bravo"})
		assert.Error(t, err)
	})
	t.Run("should panic when node node can not be added", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		assert.Panics(t, func() {
			tree.MustAdd("invalid", MyNode{"1", "Alpha"})
		})
	})
	t.Run("can add a node to zero value", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		n := MyNode{"1", "Alpha"}
		uid := tree.MustAdd(iwidget.TreeRootID, n)
		assert.Equal(t, n, node(tree, uid))
	})
	t.Run("should panic when object is invalid", func(t *testing.T) {
		var tree *iwidget.TreeData[MyNode]
		assert.PanicsWithError(t, iwidget.ErrInvalid.Error(), func() {
			tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		})
	})
}

func TestTreeData_Node(t *testing.T) {
	t.Run("should return node when it exists", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		n1 := MyNode{"1", "Alpha"}
		uid := tree.MustAdd(iwidget.TreeRootID, n1)
		n2, ok := tree.Node(uid)
		assert.True(t, ok)
		assert.Equal(t, n1, n2)

	})
	t.Run("should report when node does not exist", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		_, ok := tree.Node("invalid")
		assert.False(t, ok)
	})
	t.Run("the root node always exists", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		n, ok := tree.Node(iwidget.TreeRootID)
		assert.True(t, ok)
		assert.Equal(t, MyNode{}, n)
	})
}

func TestTreeData_Parent(t *testing.T) {
	t.Run("can return parent of a node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		uid1 := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		uid2 := tree.MustAdd(uid1, MyNode{"2", "Bravo"})
		p, ok := tree.Parent(uid2)
		assert.True(t, ok)
		assert.Equal(t, uid1, p)
	})
	t.Run("the parent of a top node is the root node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		uid := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		p, ok := tree.Parent(uid)
		assert.True(t, ok)
		assert.Equal(t, iwidget.TreeRootID, p)
	})
	t.Run("can report when a parent does not exist", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		_, ok := tree.Parent("1")
		assert.False(t, ok)
	})
	t.Run("the root node has no parents", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		_, ok := tree.Parent(iwidget.TreeRootID)
		assert.False(t, ok)
	})
}

func TestTreeData_Path(t *testing.T) {
	t.Run("should return path for an existing node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		uid1 := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		uid2 := tree.MustAdd(uid1, MyNode{"2", "Bravo"})
		uid3 := tree.MustAdd(uid2, MyNode{"3", "Charlie"})
		p := tree.Path(uid3)
		assert.Equal(t, []widget.TreeNodeID{uid1, uid2}, p)
	})
	t.Run("should return empty array for root node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		p := tree.Path(iwidget.TreeRootID)
		assert.Equal(t, []widget.TreeNodeID{}, p)
	})
	t.Run("should return empty array for a top node", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		uid := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		p := tree.Path(uid)
		assert.Equal(t, []widget.TreeNodeID{}, p)
	})
}

func TestTreeData_All(t *testing.T) {
	t.Run("returns list of all nodes", func(t *testing.T) {
		var nodes iwidget.TreeData[MyNode]
		branch := MyNode{"1", "Root"}
		nodes.MustAdd(iwidget.TreeRootID, branch)
		alpha := MyNode{"2", "Alpha"}
		nodes.Add(branch.UID(), alpha)
		bravo := MyNode{"3", "Bravo"}
		nodes.Add(branch.UID(), bravo)
		got := slices.Collect(nodes.All())
		want := []MyNode{branch, alpha, bravo}
		assert.ElementsMatch(t, want, got)
	})
}

func TestTreeData_Clear(t *testing.T) {
	t.Run("can clear tree with nodes", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		tree.Clear()
		assert.True(t, tree.IsEmpty())
	})
	t.Run("can clear empty tree", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		tree.Clear()
		assert.True(t, tree.IsEmpty())
	})
	t.Run("can clear nill tree", func(t *testing.T) {
		var tree *iwidget.TreeData[MyNode]
		tree.Clear()
	})
}

func TestTreeData_Clone(t *testing.T) {
	t.Run("can clone a td object", func(t *testing.T) {
		// given
		var td iwidget.TreeData[MyNode]
		root := MyNode{"1", "Root"}
		td.MustAdd(iwidget.TreeRootID, root)
		alpha := MyNode{"2", "Alpha"}
		td.Add(root.UID(), alpha)
		bravo := MyNode{"3", "Bravo"}
		td.Add(root.UID(), bravo)
		// when
		got := td.Clone()
		// then
		assert.True(t, got.Equal(td), "got %q, wanted %q", got, td)
	})
	t.Run("can clone a empty td object", func(t *testing.T) {
		// given
		var td iwidget.TreeData[MyNode]
		// when
		got := td.Clone()
		// then
		assert.True(t, got.Equal(td), "got %q, wanted %q", got, td)
	})
}

func TestTreeData_Equal(t *testing.T) {
	t.Run("report equal", func(t *testing.T) {
		var t1, t2 iwidget.TreeData[MyNode]
		sub1 := t1.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root"})
		t1.Add(sub1, MyNode{"2", "Alpha"})
		t1.Add(sub1, MyNode{"3", "Bravo"})
		sub2 := t2.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root"})
		t2.Add(sub2, MyNode{"2", "Alpha"})
		t2.Add(sub2, MyNode{"3", "Bravo"})
		assert.True(t, t1.Equal(t2), "got %q, wanted %q", t, t2)
	})
	t.Run("report not equal", func(t *testing.T) {
		var n1, n2 iwidget.TreeData[MyNode]
		sub1 := n1.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root"})
		n1.Add(sub1, MyNode{"2", "Alpha"})
		n1.Add(sub1, MyNode{"3", "Bravo"})
		sub2 := n2.MustAdd(iwidget.TreeRootID, MyNode{"1", "Root"})
		n2.Add(sub2, MyNode{"2", "Alpha"})
		assert.False(t, n1.Equal(n2))
	})
}

func TestTreeData_Size(t *testing.T) {
	t.Run("can return size of tree with nodes", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
		got := tree.Size()
		assert.Equal(t, 2, got)
	})
	t.Run("can return size of zero tree", func(t *testing.T) {
		var tree iwidget.TreeData[MyNode]
		got := tree.Size()
		assert.Equal(t, 1, got)
	})
}

func TestTreeData_String(t *testing.T) {
	var tree iwidget.TreeData[MyNode]
	uid := tree.MustAdd(iwidget.TreeRootID, MyNode{"1", "Alpha"})
	tree.MustAdd(uid, MyNode{"2", "Bravo"})
	s := fmt.Sprint(tree)
	assert.Equal(t, "{nodes map[1:{ID:1 Value:Alpha} 2:{ID:2 Value:Bravo}], children: map[:[1] 1:[2]]}", s)
}

// node returns a node from a tree or a zero node if not found.
// This is a helper for simpler test code.
func node[T iwidget.TreeNode](td iwidget.TreeData[T], uid widget.TreeNodeID) T {
	var zero T
	v, ok := td.Node(uid)
	if !ok {
		return zero
	}
	return v
}
