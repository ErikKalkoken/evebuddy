package fynetree_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	"github.com/stretchr/testify/assert"
)

func TestFyneTree(t *testing.T) {
	t.Parallel()
	t.Run("can create tree", func(t *testing.T) {
		tree := fynetree.New[string]()
		n1 := tree.MustAdd("", "Alpha")
		n11 := tree.MustAdd(n1, "one")
		n12 := tree.MustAdd(n1, "two")
		n2 := tree.MustAdd("", "Bravo")
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
		for _, n := range nodes {
			v := tree.MustValue(n.uid)
			assert.Equal(t, n.value, v)
			assert.Equal(t, n.isBranch, tree.IsBranch(n.uid))
		}
	})
	t.Run("can clear all nodes", func(t *testing.T) {
		tree := fynetree.New[string]()
		n := tree.MustAdd("", "Alpha")
		tree.MustAdd(n, "one")
		tree.MustAdd(n, "two")
		tree.Clear()
		assert.Equal(t, 0, tree.Size())
	})
}

func TestFyneTreeAdd(t *testing.T) {
	t.Parallel()
	t.Run("can add a node 1", func(t *testing.T) {
		tree := fynetree.New[string]()
		uid, err := tree.Add("", "Alpha")
		if assert.NoError(t, err) {
			assert.Equal(t, "Alpha", tree.MustValue(uid))
		}
	})
	t.Run("can add a node 2", func(t *testing.T) {
		tree := fynetree.New[string]()
		uid := tree.MustAdd("", "Alpha")
		assert.Equal(t, "Alpha", tree.MustValue(uid))
	})
	t.Run("should return error when node node can not be added", func(t *testing.T) {
		tree := fynetree.New[string]()
		_, err := tree.Add("invalid", "Alpha")
		assert.Error(t, err)
	})
	t.Run("should panic when node node can not be added", func(t *testing.T) {
		tree := fynetree.New[string]()
		assert.Panics(t, func() {
			tree.MustAdd("invalid", "Alpha")
		})
	})
}

func TestFyneTreeValue(t *testing.T) {
	t.Parallel()
	t.Run("can return value of a node", func(t *testing.T) {
		tree := fynetree.New[string]()
		uid := tree.MustAdd("", "Alpha")
		v := tree.MustValue(uid)
		assert.Equal(t, "Alpha", v)
	})
	t.Run("should return value when node exists", func(t *testing.T) {
		tree := fynetree.New[string]()
		uid := tree.MustAdd("", "Alpha")
		v := tree.ValueWithFallback(uid, "Fallback")
		assert.Equal(t, "Alpha", v)
	})
	t.Run("should return fallback when a node does not exist", func(t *testing.T) {
		tree := fynetree.New[string]()
		v := tree.ValueWithFallback("invalid", "Fallback")
		assert.Equal(t, "Fallback", v)
	})
	t.Run("should return value when a node exists", func(t *testing.T) {
		tree := fynetree.New[string]()
		uid := tree.MustAdd("", "Alpha")
		v, err := tree.Value(uid)
		if assert.NoError(t, err) {
			assert.Equal(t, "Alpha", v)
		}
	})
	t.Run("should return error when node does not exist", func(t *testing.T) {
		tree := fynetree.New[string]()
		_, err := tree.Value("invalid")
		assert.Error(t, err)
	})
	t.Run("should panic when node does not exist", func(t *testing.T) {
		tree := fynetree.New[string]()
		assert.Panics(t, func() {
			tree.MustValue("invalid")

		})
	})
}
