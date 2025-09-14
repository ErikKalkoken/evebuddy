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
	locationID   int64
	quantity     int32
	name         string
	itemID       int64
	locationFlag app.LocationFlag
}

func createCharacterAsset(arg characterAssetParams) *app.CharacterAsset {
	if arg.quantity == 0 {
		arg.quantity = 1
	}
	if arg.itemID == 0 {
		arg.itemID = sequence.Add(1)
	}
	if arg.locationFlag == app.FlagUndefined {
		arg.locationFlag = app.FlagHangar
	}
	return &app.CharacterAsset{
		ItemID:       arg.itemID,
		LocationID:   arg.locationID,
		Quantity:     arg.quantity,
		LocationFlag: arg.locationFlag,
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
	t.Run("can return asset nodes by item IDs", func(t *testing.T) {
		cases := []struct {
			itemID int64
			found  bool
		}{
			{a1.ItemID, true},
			{a2.ItemID, true},
			{a11.ItemID, true},
			{a111.ItemID, true},
			{a1111.ItemID, true},
			{a3.ItemID, true},
			{a31.ItemID, true},
			{666, false},
		}
		for _, tc := range cases {
			got, found := ac.Asset(tc.itemID)
			if tc.found {
				assert.True(t, found)
				assert.Equal(t, tc.itemID, got.Asset.ItemID)
			} else {
				assert.False(t, found)
				assert.Nil(t, got)
			}
		}
	})
}

func TestAssetCollection_ReturnEmptyWhenNotInitialized(t *testing.T) {
	var ac assetcollection.AssetCollection
	_, x1 := ac.AssetParentLocation(99)
	assert.False(t, x1)
	_, x2 := ac.Asset(99)
	assert.False(t, x2)
	_, x3 := ac.Location(99)
	assert.False(t, x3)
	x4 := ac.Locations()
	assert.Empty(t, x4)
}

func TestAssetCollection_ItemCount(t *testing.T) {
	const location1 = 100000
	a1 := createCharacterAsset(characterAssetParams{locationID: location1})
	a11 := createCharacterAsset(characterAssetParams{locationID: a1.ItemID})
	a111 := createCharacterAsset(characterAssetParams{locationID: a11.ItemID, quantity: 3})
	a1111 := createCharacterAsset(characterAssetParams{
		locationID:   a111.ItemID,
		locationFlag: app.FlagHiSlot0,
	})
	a1112 := createCharacterAsset(characterAssetParams{
		locationID:   a111.ItemID,
		locationFlag: app.FlagSpecializedFuelBay,
	})
	a1113 := createCharacterAsset(characterAssetParams{
		locationID:   a111.ItemID,
		locationFlag: app.FlagDroneBay,
	})
	a1114 := createCharacterAsset(characterAssetParams{
		locationID:   a111.ItemID,
		locationFlag: app.FlagFighterBay,
	})
	a1115 := createCharacterAsset(characterAssetParams{
		locationID:   a111.ItemID,
		locationFlag: app.FlagCargo,
	})
	a2 := createCharacterAsset(characterAssetParams{locationID: location1})
	assets := []*app.CharacterAsset{a1, a2, a11, a111, a1111, a1112, a1113, a1114, a1115}
	loc1 := &app.EveLocation{ID: location1, Name: "Alpha"}
	locations := []*app.EveLocation{loc1}
	ac := assetcollection.New(assets, locations)
	t.Run("can calculate item count for a location", func(t *testing.T) {
		ln, found := ac.Location(location1)
		if !found {
			t.Fatal("could not find location")
		}
		assert.Equal(t, 6, ln.ItemCountFiltered())
	})
	t.Run("can calculate item count for an asset", func(t *testing.T) {
		an, found := ac.Asset(a1.ItemID)
		if !assert.True(t, found) {
			t.Fatal()
		}
		assert.Equal(t, 5, an.ItemCountFiltered())
	})
}
