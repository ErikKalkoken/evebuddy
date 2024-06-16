package assettree

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestAssetTree(t *testing.T) {
	a1 := &app.CharacterAsset{ItemID: 1, LocationID: 100000}
	a11 := &app.CharacterAsset{ItemID: 11, LocationID: 1}
	a111 := &app.CharacterAsset{ItemID: 111, LocationID: 11}
	a1111 := &app.CharacterAsset{ItemID: 1111, LocationID: 111}
	a2 := &app.CharacterAsset{ItemID: 2, LocationID: 100000}
	a3 := &app.CharacterAsset{ItemID: 3, LocationID: 101000}
	a31 := &app.CharacterAsset{ItemID: 31, LocationID: 3}
	assets := []*app.CharacterAsset{a1, a2, a11, a111, a3, a31, a1111}
	loc1 := &app.EveLocation{ID: 100000, Name: "Alpha"}
	loc2 := &app.EveLocation{ID: 101000, Name: "Bravo"}
	locations := []*app.EveLocation{loc1, loc2}
	at := New(assets, locations)
	t.Run("can create tree from character assets", func(t *testing.T) {
		assert.Len(t, at.lns, 2)
		assert.Len(t, at.lns[100000].nodes, 2)
		assert.Len(t, at.lns[100000].nodes[a1.ItemID].nodes, 1)
		assert.Len(t, at.lns[100000].nodes[a1.ItemID].nodes[a11.ItemID].nodes, 1)
		assert.Len(t, at.lns[100000].nodes[a2.ItemID].nodes, 0)
		assert.Len(t, at.lns[101000].nodes, 1)
		assert.Len(t, at.lns[101000].nodes[a3.ItemID].nodes, 1)
		assert.Len(t, at.lns[101000].nodes[a31.ItemID].nodes, 0)
	})
	t.Run("can return parent location for assets", func(t *testing.T) {
		cases := []struct {
			id       int64
			found    bool
			location *app.EveLocation
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
