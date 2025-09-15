package assetcollection_test

import (
	"slices"
	"sync/atomic"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
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
		locationID1 = 100000
		locationID2 = 101000
	)
	a1 := createCharacterAsset(characterAssetParams{locationID: locationID1})
	a11 := createCharacterAsset(characterAssetParams{locationID: a1.ItemID})
	a111 := createCharacterAsset(characterAssetParams{locationID: a11.ItemID, quantity: 3})
	a1111 := createCharacterAsset(characterAssetParams{locationID: a111.ItemID})
	a2 := createCharacterAsset(characterAssetParams{locationID: locationID1})
	a3 := createCharacterAsset(characterAssetParams{locationID: locationID2})
	a31 := createCharacterAsset(characterAssetParams{locationID: a3.ItemID})
	assets := []*app.CharacterAsset{a1, a2, a11, a111, a3, a31, a1111}
	loc1 := &app.EveLocation{ID: locationID1, Name: "Alpha"}
	loc2 := &app.EveLocation{ID: locationID2, Name: "Bravo"}
	locations := []*app.EveLocation{loc1, loc2}
	ac := assetcollection.New(assets, locations)
	t.Run("can create tree from character assets", func(t *testing.T) {
		locations := ac.Locations()
		assert.Len(t, locations, 2)
		for _, l := range locations {
			if l.Location.ID == locationID1 {
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
			if l.Location.ID == locationID2 {
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
		ln1, _ := ac.Location(loc1.ID)
		ln2, _ := ac.Location(loc2.ID)
		cases := []struct {
			itemID   int64
			found    bool
			location *assetcollection.LocationNode
		}{
			{a1.ItemID, true, ln1},
			{a2.ItemID, true, ln1},
			{a11.ItemID, true, ln1},
			{a111.ItemID, true, ln1},
			{a1111.ItemID, true, ln1},
			{a3.ItemID, true, ln2},
			{a31.ItemID, true, ln2},
			{666, false, nil},
		}
		for _, tc := range cases {
			got, found := ac.AssetLocation(tc.itemID)
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

func TestAssetCollection_Walk(t *testing.T) {
	const locationID = 100000
	a1 := createCharacterAsset(characterAssetParams{locationID: locationID})
	a11 := createCharacterAsset(characterAssetParams{locationID: a1.ItemID})
	a111 := createCharacterAsset(characterAssetParams{locationID: a11.ItemID})
	a1111 := createCharacterAsset(characterAssetParams{
		locationID: a111.ItemID,
	})
	a1112 := createCharacterAsset(characterAssetParams{
		locationID: a111.ItemID,
	})
	assets := []*app.CharacterAsset{a1, a11, a111, a1111, a1112}
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	ac := assetcollection.New(assets, locations)
	t.Run("can walk branch", func(t *testing.T) {
		an, _ := ac.Asset(a1.ItemID)
		s := an.All()
		got := slices.Collect(xiter.MapSlice(s, func(x *assetcollection.AssetNode) int64 {
			return x.Asset.ItemID
		}))
		want := slices.Collect(xiter.MapSlice(assets, func(x *app.CharacterAsset) int64 {
			return x.ItemID
		}))
		assert.ElementsMatch(t, want, got)
	})
}

func TestAssetCollection_ReturnEmptyWhenNotInitialized(t *testing.T) {
	var ac assetcollection.AssetCollection
	_, x1 := ac.AssetLocation(99)
	assert.False(t, x1)
	_, x2 := ac.Asset(99)
	assert.False(t, x2)
	_, x3 := ac.Location(99)
	assert.False(t, x3)
	x4 := ac.Locations()
	assert.Empty(t, x4)
}

func TestAssetCollection_ItemCount(t *testing.T) {
	const locationID = 100000
	a1 := createCharacterAsset(characterAssetParams{locationID: locationID})
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
	a2 := createCharacterAsset(characterAssetParams{locationID: locationID})
	assets := []*app.CharacterAsset{a1, a2, a11, a111, a1111, a1112, a1113, a1114, a1115}
	loc1 := &app.EveLocation{ID: locationID, Name: "Alpha"}
	locations := []*app.EveLocation{loc1}
	ac := assetcollection.New(assets, locations)
	t.Run("can calculate item count for a location", func(t *testing.T) {
		ln, found := ac.Location(locationID)
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
