package widget

import (
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MyNode2 struct {
	Text string
}

func TestTreeData2_CanCreateFullTree(t *testing.T) {
	var tree TreeData2[MyNode2]
	alpha := &MyNode2{"Alpha"}
	tree.Add(nil, alpha, true)
	n11 := &MyNode2{"one"}
	tree.Add(alpha, n11, false)
	n12 := &MyNode2{"two"}
	tree.Add(alpha, n12, false)
	bravo := &MyNode2{"Bravo"}
	tree.Add(nil, bravo, false)
	assert.Equal(t, 4, tree.Size())
	got1 := tree.Children(alpha)
	assert.Equal(t, []*MyNode2{n11, n12}, got1)
	got2 := tree.Children(bravo)
	assert.Len(t, got2, 0)
	nodes := []struct {
		node     *MyNode2
		value    string
		isBranch bool
	}{
		{alpha, "Alpha", true},
		{n11, "one", false},
		{n12, "two", false},
		{bravo, "Bravo", false},
	}
	for _, tc := range nodes {
		uid, ok := tree.UID(tc.node)
		require.True(t, ok)
		n, ok := tree.Node(uid)
		require.True(t, ok)
		assert.Equal(t, n, tc.node)
	}
}

func TestTreeData2_ChildUIDs(t *testing.T) {
	t.Run("can return child UIDs of existing node", func(t *testing.T) {
		var tree TreeData2[MyNode2]
		sub1 := &MyNode2{"Alpha"}
		c1 := &MyNode2{"Bravo"}
		c2 := &MyNode2{"Charlie"}
		sub2 := &MyNode2{"Delta"}
		tree.Add(nil, sub1, true)
		tree.Add(sub1, c1, true)
		tree.Add(sub1, c2, true)
		tree.Add(nil, sub2, true)
		tree.Add(sub2, &MyNode2{"Echo"}, true)
		uid, ok := tree.UID(sub1)
		require.True(t, ok)
		got := tree.children[uid]
		c1UID, ok := tree.UID(c1)
		require.True(t, ok)
		c2UID, ok := tree.UID(c2)
		require.True(t, ok)
		want := []widget.TreeNodeID{c1UID, c2UID}
		assert.ElementsMatch(t, want, got)
	})
}

func TestTreeData2_Clone(t *testing.T) {
	t.Run("can clone a td object", func(t *testing.T) {
		// given
		var td TreeData2[MyNode2]
		root := &MyNode2{"Root"}
		td.Add(nil, root, true)
		alpha := &MyNode2{"Alpha"}
		td.Add(root, alpha, false)
		bravo := &MyNode2{"Bravo"}
		td.Add(root, bravo, false)
		// when
		got := td.Clone()
		// then
		TreeDataEqual(t, td, got)
	})
	t.Run("can clone a empty td object", func(t *testing.T) {
		// given
		td := newTreeData2[MyNode2]()
		// when
		got := td.Clone()
		// then
		TreeDataEqual(t, td, got)
	})
}

func TreeDataEqual[T any](t *testing.T, want, got TreeData2[T]) {
	t.Helper()
	assert.Equal(t, want.children, got.children)
	assert.Equal(t, want.id, got.id)
	assert.Equal(t, want.isBranchNode, got.isBranchNode)
	assert.Equal(t, want.nodes, got.nodes)
	assert.Equal(t, want.parents, got.parents)
	assert.Equal(t, want.uidLookup, got.uidLookup)
}
