package ui

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestAssetTree(t *testing.T) {
	a1 := &model.CharacterAsset{ItemID: 1, LocationID: 100000}
	a11 := &model.CharacterAsset{ItemID: 11, LocationID: 1}
	a111 := &model.CharacterAsset{ItemID: 111, LocationID: 11}
	a1111 := &model.CharacterAsset{ItemID: 1111, LocationID: 111}
	a2 := &model.CharacterAsset{ItemID: 2, LocationID: 100000}
	a3 := &model.CharacterAsset{ItemID: 3, LocationID: 101000}
	a31 := &model.CharacterAsset{ItemID: 31, LocationID: 3}
	assets := []*model.CharacterAsset{a1, a2, a11, a111, a3, a31, a1111}
	t.Run("can create tree from character assets", func(t *testing.T) {
		nodes := NewAssetTree(assets)
		assert.Len(t, nodes, 3)
		assert.Len(t, nodes[a1.ItemID].children, 1)
		assert.Len(t, nodes[a1.ItemID].children[a11.ItemID].children, 1)
		assert.Len(t, nodes[a2.ItemID].children, 0)
		assert.Len(t, nodes[a3.ItemID].children, 1)
		assert.Len(t, nodes[a3.ItemID].children[a31.ItemID].children, 0)
	})
	t.Run("can create location list from tree", func(t *testing.T) {
		nodes := NewAssetTree(assets)
		got := CompileAssetParentLocations(nodes)
		want := map[int64]int64{
			1:    100000,
			2:    100000,
			11:   100000,
			111:  100000,
			1111: 100000,
			3:    101000,
			31:   101000,
		}
		assert.Equal(t, want, got)
	})
}
