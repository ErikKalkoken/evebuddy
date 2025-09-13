package assetcollection_test

import (
	"sync/atomic"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/stretchr/testify/assert"
)

var sequence atomic.Int64

type characterAssetParams struct {
	locationID int64
	quantity   int32
	name       string
	itemID     int64
}

func createCharacterAsset(arg characterAssetParams) *app.CharacterAsset {
	if arg.quantity == 0 {
		arg.quantity = 1
	}
	if arg.itemID == 0 {
		arg.itemID = sequence.Add(1)
	}
	return &app.CharacterAsset{
		ItemID:     arg.itemID,
		LocationID: arg.locationID,
		Quantity:   arg.quantity,
	}
}

func TestAssetCollection(t *testing.T) {
	const (
		location1 = 100000
		location2 = 101000
	)
	a1 := createCharacterAsset(characterAssetParams{locationID: location1})
	a11 := createCharacterAsset(characterAssetParams{locationID: a1.ItemID})
	a111 := createCharacterAsset(characterAssetParams{locationID: a11.ItemID, quantity: 3})
	a1111 := createCharacterAsset(characterAssetParams{locationID: a111.ItemID})
	a2 := createCharacterAsset(characterAssetParams{locationID: location1})
	a3 := createCharacterAsset(characterAssetParams{locationID: location2})
	a31 := createCharacterAsset(characterAssetParams{locationID: a3.ItemID})
	assets := []*app.CharacterAsset{a1, a2, a11, a111, a3, a31, a1111}
	loc1 := &app.EveLocation{ID: location1, Name: "Alpha"}
	loc2 := &app.EveLocation{ID: location2, Name: "Bravo"}
	locations := []*app.EveLocation{loc1, loc2}
	ac := assetcollection.New(assets, locations)
	t.Run("can create tree from character assets", func(t *testing.T) {
		locations := ac.Locations()
		assert.Len(t, locations, 2)
		for _, l := range locations {
			if l.Location.ID == location1 {
				nodes := l.Children()
				assert.Len(t, nodes, 2)
				for _, n := range nodes {
					if n.Asset.ItemID == a1.ItemID {
						assert.Len(t, n.Children(), 1)
						if n.Asset.ItemID == a1.ItemID {
							assert.Len(t, n.Children(), 1)
							sub := n.Children()[0]
							assert.Equal(t, a11.ItemID, sub.Asset.ItemID)
							assert.Len(t, sub.Children(), 1)
						}
					}
					if n.Asset.ItemID == a2.ItemID {
						assert.Len(t, n.Children(), 0)
					}
				}
			}
			if l.Location.ID == location2 {
				nodes := l.Children()
				assert.Len(t, nodes, 1)
				sub := nodes[0]
				assert.Equal(t, a3.ItemID, sub.Asset.ItemID)
				sub2 := sub.Children()
				assert.Len(t, sub2, 1)
				assert.Equal(t, a31.ItemID, sub2[0].Asset.ItemID)
			}
		}
	})
	t.Run("can return parent location for assets", func(t *testing.T) {
		cases := []struct {
			itemID   int64
			found    bool
			location *app.EveLocation
		}{
			{a1.ItemID, true, loc1},
			{a2.ItemID, true, loc1},
			{a11.ItemID, true, loc1},
			{a111.ItemID, true, loc1},
			{a1111.ItemID, true, loc1},
			{a3.ItemID, true, loc2},
			{a31.ItemID, true, loc2},
			{666, false, nil},
		}
		for _, tc := range cases {
			got, found := ac.AssetParentLocation(tc.itemID)
			if assert.Equal(t, tc.found, found) {
				assert.Equal(t, tc.location, got)
			}
		}
	})
	t.Run("can calculate item count for location 1", func(t *testing.T) {
		ln, found := ac.Location(location1)
		if !found {
			t.Fatal("could not find location")
		}
		assert.Equal(t, 7, ln.ItemCount())
	})
	t.Run("can calculate item count for location 2", func(t *testing.T) {
		ln, found := ac.Location(location2)
		if !found {
			t.Fatal("could not find location")
		}
		assert.Equal(t, 2, ln.ItemCount())
	})
}
