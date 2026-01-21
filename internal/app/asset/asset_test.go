package asset_test

import (
	"slices"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/asset"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

var sequence atomic.Int64

type characterAssetParams struct {
	locationID   int64
	quantity     int
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
	ac := asset.NewFromCharacterAssets(assets, locations)
	t.Run("can create tree from character assets", func(t *testing.T) {
		locations := ac.LocationNodes()
		assert.Len(t, locations, 2)
		for _, l := range locations {
			if l.ID() == locationID1 {
				nodes := l.Children()
				assert.Len(t, nodes, 2)
				for _, n := range nodes {
					if n.Item().ID() == a1.ItemID {
						assert.Len(t, n.Children(), 1)
						if n.Item().ID() == a1.ItemID {
							assert.Len(t, n.Children(), 1)
							sub := n.Children()[0]
							assert.Equal(t, a11.ItemID, sub.Item().ID())
							assert.Len(t, sub.Children(), 1)
						}
					}
					if n.Item().ID() == a2.ItemID {
						assert.Len(t, n.Children(), 0)
					}
				}
			}
			if l.ID() == locationID2 {
				nodes := l.Children()
				assert.Len(t, nodes, 1)
				sub := nodes[0]
				assert.Equal(t, a3.ItemID, sub.Item().ID())
				sub2 := sub.Children()
				assert.Len(t, sub2, 1)
				assert.Equal(t, a31.ItemID, sub2[0].Item().ID())
			}
		}
	})
	t.Run("can return parent", func(t *testing.T) {
		cases := []struct {
			item           asset.Item
			isItem         bool
			parentItem     asset.Item
			parentLocation *app.EveLocation
		}{
			{a11, true, a1, nil},
			{a1, false, nil, loc1},
		}
		for _, tc := range cases {
			n, found := ac.Node(tc.item.ID())
			require.True(t, found)
			p := n.Parent()
			if !tc.isItem {
				assert.Equal(t, tc.parentLocation, p.Location())
				continue
			}
			assert.Equal(t, tc.parentItem, p.Item())
		}
	})
	t.Run("can return path", func(t *testing.T) {
		cases := []struct {
			item int64
			want []int64
		}{
			{a11.ID(), []int64{loc1.ID, a1.ID()}},
			{a1111.ID(), []int64{loc1.ID, a1.ID(), a11.ID(), a111.ID()}},
		}
		for _, tc := range cases {
			n, found := ac.Node(tc.item)
			require.True(t, found)
			path := n.Path()
			got := xslices.Map(path, func(x *asset.Node) int64 {
				return x.ID()
			})
			assert.Equal(t, tc.want, got)
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
			got, found := ac.Node(tc.itemID)
			if tc.found {
				assert.True(t, found)
				assert.Equal(t, tc.itemID, got.Item().ID())
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
	ac := asset.NewFromCharacterAssets(assets, locations)
	t.Run("can walk branch", func(t *testing.T) {
		an, _ := ac.Node(a1.ItemID)
		s := an.All()
		got := slices.Collect(xiter.MapSlice(s, func(x *asset.Node) int64 {
			return x.Item().ID()
		}))
		want := slices.Collect(xiter.MapSlice(assets, func(x *app.CharacterAsset) int64 {
			return x.ItemID
		}))
		assert.ElementsMatch(t, want, got)
	})
}

func TestAssetCollection_ReturnEmptyWhenNotInitialized(t *testing.T) {
	var ac asset.Collection
	_, x1 := ac.RootLocationNode(99)
	assert.False(t, x1)
	_, x2 := ac.Node(99)
	assert.False(t, x2)
	x4 := ac.LocationNodes()
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
	ac := asset.NewFromCharacterAssets(assets, locations)
	t.Run("can calculate item count for an asset", func(t *testing.T) {
		n, found := ac.Node(a1.ItemID)
		if !assert.True(t, found) {
			t.Fatal()
		}
		assert.Equal(t, 5, n.ItemCountFiltered())
	})
}
