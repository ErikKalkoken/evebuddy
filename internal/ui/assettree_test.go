package ui

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestAssetTree(t *testing.T) {
	a1 := &model.CharacterAsset{ID: 1, LocationID: 100}
	a2 := &model.CharacterAsset{ID: 2, LocationID: 100}
	a3 := &model.CharacterAsset{ID: 3, LocationID: 1}
	a4 := &model.CharacterAsset{ID: 4, LocationID: 3}
	a5 := &model.CharacterAsset{ID: 5, LocationID: 101}
	a6 := &model.CharacterAsset{ID: 6, LocationID: 5}
	assets := []*model.CharacterAsset{a1, a2, a3, a4, a5, a6}
	t.Run("can create tree from character assets", func(t *testing.T) {
		nodes := newAssetTree(assets)
		assert.Len(t, nodes, 3)
		assert.Len(t, nodes[a1.ID].children, 1)
		assert.Len(t, nodes[a1.ID].children[a3.ID].children, 1)
		assert.Len(t, nodes[a2.ID].children, 0)
		assert.Len(t, nodes[a5.ID].children, 1)
		assert.Len(t, nodes[a5.ID].children[a6.ID].children, 0)
	})
	t.Run("can create location list from tree", func(t *testing.T) {
		nodes := newAssetTree(assets)
		xx := collectAssetParentLocations(nodes)
		assert.Len(t, xx, 6)
		got := make(map[int64]int64)
		for _, x := range xx {
			got[x.assetID] = x.locationID
		}
		want := map[int64]int64{
			1: 100,
			2: 100,
			3: 100,
			4: 100,
			5: 101,
			6: 101,
		}
		assert.Equal(t, want, got)
	})
}
