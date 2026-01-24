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
	LocationID   int64
	Quantity     int
	Name         string
	ItemID       int64
	LocationFlag app.LocationFlag
	Type         *app.EveType
}

func createCharacterAsset(arg characterAssetParams) *app.CharacterAsset {
	if arg.Quantity == 0 {
		arg.Quantity = 1
	}
	if arg.ItemID == 0 {
		arg.ItemID = sequence.Add(1)
	}
	if arg.LocationFlag == app.FlagUndefined {
		arg.LocationFlag = app.FlagHangar
	}
	return &app.CharacterAsset{
		Asset: app.Asset{
			ItemID:       arg.ItemID,
			LocationFlag: arg.LocationFlag,
			LocationID:   arg.LocationID,
			Name:         arg.Name,
			Quantity:     arg.Quantity,
			Type:         arg.Type,
		},
	}
}

func TestCollection(t *testing.T) {
	t.Skip("need to be updated after recent changes")
	const (
		locationID1 = 100000
		locationID2 = 101000
	)
	a1 := createCharacterAsset(characterAssetParams{LocationID: locationID1})
	a11 := createCharacterAsset(characterAssetParams{LocationID: a1.ItemID})
	a111 := createCharacterAsset(characterAssetParams{LocationID: a11.ItemID, Quantity: 3})
	a1111 := createCharacterAsset(characterAssetParams{LocationID: a111.ItemID})
	a2 := createCharacterAsset(characterAssetParams{LocationID: locationID1})
	a3 := createCharacterAsset(characterAssetParams{LocationID: locationID2})
	a31 := createCharacterAsset(characterAssetParams{LocationID: a3.ItemID})
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
					if n.ID() == a1.ItemID {
						assert.Len(t, n.Children(), 1)
						if n.ID() == a1.ItemID {
							assert.Len(t, n.Children(), 1)
							sub := n.Children()[0]
							assert.Equal(t, a11.ItemID, sub.ID())
							assert.Len(t, sub.Children(), 1)
						}
					}
					if n.ID() == a2.ItemID {
						assert.Len(t, n.Children(), 0)
					}
				}
			}
			if l.ID() == locationID2 {
				nodes := l.Children()
				assert.Len(t, nodes, 1)
				sub := nodes[0]
				assert.Equal(t, a3.ItemID, sub.ID())
				sub2 := sub.Children()
				assert.Len(t, sub2, 1)
				assert.Equal(t, a31.ItemID, sub2[0].ID())
			}
			asset.PrintTree(l)
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
				assert.Equal(t, tc.parentLocation, p.MustLocation())
				continue
			}
			assert.Equal(t, tc.parentItem, p.MustCharacterAsset())
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
				assert.Equal(t, tc.itemID, got.ID())
			} else {
				assert.False(t, found)
				assert.Nil(t, got)
			}
		}
	})
}

func TestCollection_All(t *testing.T) {
	const locationID = 100000
	a1 := createCharacterAsset(characterAssetParams{LocationID: locationID})
	a11 := createCharacterAsset(characterAssetParams{LocationID: a1.ItemID})
	a111 := createCharacterAsset(characterAssetParams{LocationID: a11.ItemID})
	a1111 := createCharacterAsset(characterAssetParams{
		LocationID: a111.ItemID,
	})
	a1112 := createCharacterAsset(characterAssetParams{
		LocationID: a111.ItemID,
	})
	assets := []*app.CharacterAsset{a1, a11, a111, a1111, a1112}
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	ac := asset.NewFromCharacterAssets(assets, locations)
	t.Run("can walk branch", func(t *testing.T) {
		an, _ := ac.Node(a1.ItemID)
		s := an.All()
		got := slices.Collect(xiter.MapSlice(s, func(x *asset.Node) int64 {
			return x.ID()
		}))
		want := slices.Collect(xiter.MapSlice(assets, func(x *app.CharacterAsset) int64 {
			return x.ItemID
		}))
		assert.ElementsMatch(t, want, got)
	})
}

func TestCollection_ReturnEmptyWhenNotInitialized(t *testing.T) {
	var ac asset.Collection
	_, x1 := ac.RootLocationNode(99)
	assert.False(t, x1)
	_, x2 := ac.Node(99)
	assert.False(t, x2)
	x4 := ac.LocationNodes()
	assert.Empty(t, x4)
}

func TestCollection_ItemCount(t *testing.T) {
	const locationID = 100000
	a1 := createCharacterAsset(characterAssetParams{LocationID: locationID})
	a11 := createCharacterAsset(characterAssetParams{LocationID: a1.ItemID})
	a111 := createCharacterAsset(characterAssetParams{LocationID: a11.ItemID, Quantity: 3})
	a1111 := createCharacterAsset(characterAssetParams{
		LocationID:   a111.ItemID,
		LocationFlag: app.FlagHiSlot0,
	})
	a1112 := createCharacterAsset(characterAssetParams{
		LocationID:   a111.ItemID,
		LocationFlag: app.FlagSpecializedFuelBay,
	})
	a1113 := createCharacterAsset(characterAssetParams{
		LocationID:   a111.ItemID,
		LocationFlag: app.FlagDroneBay,
	})
	a1114 := createCharacterAsset(characterAssetParams{
		LocationID:   a111.ItemID,
		LocationFlag: app.FlagFighterBay,
	})
	a1115 := createCharacterAsset(characterAssetParams{
		LocationID:   a111.ItemID,
		LocationFlag: app.FlagCargo,
	})
	a2 := createCharacterAsset(characterAssetParams{LocationID: locationID})
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

func TestCollection_CustomNodes(t *testing.T) {
	const locationID = 100000
	ship1 := createCharacterAsset(characterAssetParams{
		ItemID:     1,
		LocationID: locationID,
		Type: &app.EveType{
			Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}},
			Name:  "Merlin",
		},
	})
	drone := createCharacterAsset(characterAssetParams{
		ItemID:       2,
		LocationID:   ship1.ItemID,
		LocationFlag: app.FlagDroneBay,
		Type: &app.EveType{
			Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryDrone}},
			Name:  "Hobgoblin",
		},
	})
	mineral1 := createCharacterAsset(characterAssetParams{
		ItemID:     3,
		LocationID: locationID,
		Type: &app.EveType{
			Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryMineral}},
			Name:  "Tritanium",
		},
	})
	ship2 := createCharacterAsset(characterAssetParams{
		ItemID:     4,
		LocationID: locationID,
		Type: &app.EveType{
			Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}},
			Name:  "Tristan",
		},
	})
	office := createCharacterAsset(characterAssetParams{
		ItemID:       5,
		LocationID:   locationID,
		LocationFlag: app.FlagOfficeFolder,
		Type: &app.EveType{
			ID:    27,
			Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryStation}},
			Name:  "Office",
		},
	})
	mineral2 := createCharacterAsset(characterAssetParams{
		ItemID:       6,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG1,
		Type: &app.EveType{
			Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryMineral}},
			Name:  "Tritanium2",
		},
	})
	assets := []*app.CharacterAsset{ship1, mineral1, drone, ship2, mineral2, office}
	loc := &app.EveLocation{ID: locationID, Name: "Alpha"}
	locations := []*app.EveLocation{loc}
	ac := asset.NewFromCharacterAssets(assets, locations)
	tree := ac.LocationNodes()[0]

	makePath := func(itemID int64) []string {
		return xslices.Map(ac.MustNode(itemID).Path(), func(x *asset.Node) string {
			return x.DisplayName()
		})
	}

	assert.Equal(t, 3, len(tree.Children()))
	assert.Equal(t, []string{"Alpha", "Item Hangar"}, makePath(mineral1.ItemID))
	assert.Equal(t, []string{"Alpha", "Ship Hangar", "Merlin", "Drone Bay"}, makePath(drone.ItemID))
	assert.Equal(t, []string{"Alpha", "Ship Hangar"}, makePath(ship2.ItemID))
	assert.Equal(t, []string{"Alpha", "Office", "1st Division"}, makePath(mineral2.ItemID))
	asset.PrintTree(tree)
	// assert.Fail(t, "stop")
}

func TestUpdateItemCounts(t *testing.T) {
	const locationID = 42
	a := createCharacterAsset(characterAssetParams{
		Name:       "a",
		Quantity:   3,
		LocationID: locationID,
	})
	b := createCharacterAsset(characterAssetParams{
		Name:       "b",
		Quantity:   2,
		LocationID: a.ItemID,
	})
	c := createCharacterAsset(characterAssetParams{
		Name:       "c",
		Quantity:   3,
		LocationID: locationID,
	})
	d := createCharacterAsset(characterAssetParams{
		Name:       "d",
		Quantity:   4,
		LocationID: a.ItemID,
	})
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	assets := []*app.CharacterAsset{a, b, c, d}
	ac := asset.NewFromCharacterAssets(assets, locations)
	root := ac.LocationNodes()[0]

	assert.Equal(t, 1, root.ItemCount.ValueOrZero())
	assert.Equal(t, 2, ac.MustNode(a.ItemID).ItemCount.ValueOrZero())
	assert.Equal(t, 0, ac.MustNode(b.ItemID).ItemCount.ValueOrZero())

	asset.PrintTree(root)
	// assert.Fail(t, "")
}
