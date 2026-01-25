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

func TritaniumType() *app.EveType {
	return &app.EveType{
		ID:    34,
		Group: &app.EveGroup{ID: 18, Category: &app.EveCategory{ID: app.EveCategoryMineral}},
		Name:  "Tritanium",
	}
}

func OfficeType() *app.EveType {
	return &app.EveType{
		ID:    27,
		Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryStation}},
		Name:  "Office",
	}
}

func CargoContainerType() *app.EveType {
	return &app.EveType{
		ID:    3293,
		Group: &app.EveGroup{ID: 12, Category: &app.EveCategory{ID: 2}},
		Name:  "Medium Standard Container",
	}
}

type characterAssetParams struct {
	IsSingleton  bool
	ItemID       int64
	LocationFlag app.LocationFlag
	LocationID   int64
	LocationType app.LocationType
	Name         string
	Quantity     int
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
	if arg.LocationType == app.TypeUndefined {
		arg.LocationType = app.TypeItem
	}
	if arg.Type == nil {
		arg.Type = TritaniumType()
	}
	return &app.CharacterAsset{
		Asset: app.Asset{
			IsSingleton:  arg.IsSingleton,
			ItemID:       arg.ItemID,
			LocationFlag: arg.LocationFlag,
			LocationID:   arg.LocationID,
			LocationType: arg.LocationType,
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
		locations := ac.Trees()
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
	x4 := ac.Trees()
	assert.Empty(t, x4)
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
		Type:         OfficeType(),
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
	tree := ac.Trees()[0]

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

func TestCollection_Impounded(t *testing.T) {
	const locationID = 60007927
	office := createCharacterAsset(characterAssetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   locationID,
		LocationFlag: app.FlagImpounded,
		LocationType: app.TypeStation,
		Type:         OfficeType(),
	})
	item1 := createCharacterAsset(characterAssetParams{
		Quantity:     99,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG1,
		LocationType: app.TypeItem,
		Type:         TritaniumType(),
	})
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	assets := []*app.CharacterAsset{office, item1}
	ac := asset.NewFromCharacterAssets(assets, locations)
	root := ac.Trees()[0]

	bn := ac.MustNode(item1.ItemID)
	path := xslices.Map(bn.Path(), func(x *asset.Node) string {
		return x.DisplayName()
	})
	assert.Equal(t, []string{"Alpha", "Impounded", "Office", "1st Division"}, path)

	asset.PrintTree(root)
	// assert.Fail(t, "STOP")
}

func TestCollection_Offices(t *testing.T) {
	const locationID = 60007927
	office := createCharacterAsset(characterAssetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   locationID,
		LocationFlag: app.FlagOfficeFolder,
		LocationType: app.TypeStation,
		Type:         OfficeType(),
	})
	item1 := createCharacterAsset(characterAssetParams{
		Quantity:     99,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG1,
		LocationType: app.TypeItem,
		Type:         TritaniumType(),
	})
	item2 := createCharacterAsset(characterAssetParams{
		Quantity:     33,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG2,
		LocationType: app.TypeItem,
		Type:         TritaniumType(),
	})
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	assets := []*app.CharacterAsset{office, item1, item2}
	ac := asset.NewFromCharacterAssets(assets, locations)
	root := ac.Trees()[0]

	itemNode := ac.MustNode(item1.ItemID)
	path := xslices.Map(itemNode.Path(), func(x *asset.Node) string {
		return x.DisplayName()
	})
	assert.Equal(t, []string{"Alpha", "Office", "1st Division"}, path)

	officeNode := ac.MustNode(office.ItemID)
	offices := xslices.Map(officeNode.Children(), func(x *asset.Node) string {
		return x.DisplayName()
	})
	assert.Len(t, offices, 7)
	assert.ElementsMatch(t, []string{
		"1st Division",
		"2nd Division",
		"3rd Division",
		"4th Division",
		"5th Division",
		"6th Division",
		"7th Division",
	},
		offices,
	)
	asset.PrintTree(root)
	// assert.Fail(t, "STOP")
}
