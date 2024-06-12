package assettree_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/assettree"
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
	loc1 := &model.EveLocation{ID: 100000, Name: "Alpha"}
	loc2 := &model.EveLocation{ID: 101000, Name: "Bravo"}
	locations := []*model.EveLocation{loc1, loc2}
	at := assettree.New(assets, locations)
	t.Run("can create tree from character assets", func(t *testing.T) {
		assert.Len(t, at.Locations, 2)
		assert.Len(t, at.Locations[100000].Children, 2)
		assert.Len(t, at.Locations[100000].Children[a1.ItemID].Children, 1)
		assert.Len(t, at.Locations[100000].Children[a1.ItemID].Children[a11.ItemID].Children, 1)
		assert.Len(t, at.Locations[100000].Children[a2.ItemID].Children, 0)
		assert.Len(t, at.Locations[101000].Children, 1)
		assert.Len(t, at.Locations[101000].Children[a3.ItemID].Children, 1)
		assert.Len(t, at.Locations[101000].Children[a31.ItemID].Children, 0)
	})
	t.Run("can return parent location for assets", func(t *testing.T) {
		cases := []struct {
			id       int64
			found    bool
			location *model.EveLocation
		}{
			{1, true, loc1},
			{2, true, loc1},
			{11, true, loc1},
			{111, true, loc1},
			{1111, true, loc1},
			{3, true, loc2},
			{31, true, loc2},
			{4, false, nil},
		}
		for _, tc := range cases {
			got, found := at.AssetParentLocation(tc.id)
			if assert.Equal(t, tc.found, found) {
				assert.Equal(t, tc.location, got)
			}
		}
	})
}
