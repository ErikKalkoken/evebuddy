package assetcollection_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/stretchr/testify/assert"
)

func TestAssetCollection(t *testing.T) {
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
	ac := assetcollection.New(assets, locations)
	t.Run("can create tree from character assets", func(t *testing.T) {
		locations := ac.Locations()
		assert.Len(t, locations, 2)
		for _, l := range locations {
			if l.Location.ID == 100000 {
				nodes := l.Nodes()
				assert.Len(t, nodes, 2)
				for _, n := range nodes {
					if n.Asset.ItemID == a1.ItemID {
						assert.Len(t, n.Nodes(), 1)
						if n.Asset.ItemID == a1.ItemID {
							assert.Len(t, n.Nodes(), 1)
							sub := n.Nodes()[0]
							assert.Equal(t, a11.ItemID, sub.Asset.ItemID)
							assert.Len(t, sub.Nodes(), 1)
						}
					}
					if n.Asset.ItemID == a2.ItemID {
						assert.Len(t, n.Nodes(), 0)
					}
				}
			}
			if l.Location.ID == 101000 {
				nodes := l.Nodes()
				assert.Len(t, nodes, 1)
				sub := nodes[0]
				assert.Equal(t, a3.ItemID, sub.Asset.ItemID)
				sub2 := sub.Nodes()
				assert.Len(t, sub2, 1)
				assert.Equal(t, a31.ItemID, sub2[0].Asset.ItemID)
			}
		}
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
			got, found := ac.AssetParentLocation(tc.id)
			if assert.Equal(t, tc.found, found) {
				assert.Equal(t, tc.location, got)
			}
		}
	})
}
