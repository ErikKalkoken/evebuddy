package asset_test

import (
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/asset"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCollection(t *testing.T) {
	const (
		alphaID = 100000
		bravoID = 101000
	)
	a1 := createCharacterAsset(assetParams{LocationID: alphaID, Type: cargoContainerType()})
	a2 := createCharacterAsset(assetParams{LocationID: a1.ItemID, Type: cargoContainerType()})
	a3 := createCharacterAsset(assetParams{LocationID: a2.ItemID, Type: cargoContainerType()})
	a4 := createCharacterAsset(assetParams{LocationID: a3.ItemID, Quantity: 3})
	a5 := createCharacterAsset(assetParams{LocationID: alphaID})
	b1 := createCharacterAsset(assetParams{LocationID: bravoID, Type: cargoContainerType()})
	b2 := createCharacterAsset(assetParams{LocationID: b1.ItemID})
	assets := []*app.CharacterAsset{a1, a5, a2, a3, b1, b2, a4}
	loc1 := &app.EveLocation{ID: alphaID, Name: "Alpha"}
	loc2 := &app.EveLocation{ID: bravoID, Name: "Bravo"}
	locations := []*app.EveLocation{loc1, loc2}
	ac := asset.NewFromCharacterAssets(assets, locations)

	t.Run("can create trees from character assets", func(t *testing.T) {
		assert.Len(t, ac.Trees(), 2)

		_, ok := ac.LocationTree(loc1.ID)
		require.True(t, ok)
		assert.Equal(t, []string{"Alpha", "Item Hangar", "Container", "Container", "Container"}, makeNamesPath(ac, a4))
		assert.Equal(t, []string{"Alpha", "Item Hangar"}, makeNamesPath(ac, a5))

		_, ok = ac.LocationTree(loc2.ID)
		require.True(t, ok)
		assert.Equal(t, []string{"Bravo", "Item Hangar", "Container"}, makeNamesPath(ac, b2))

		printTrees(ac)
		// t.Fail()

	})
	t.Run("can return path", func(t *testing.T) {
		cases := []struct {
			item asset.Item
			want []string
		}{
			{a2, []string{"Alpha", "Item Hangar", "Container"}},
			{a4, []string{"Alpha", "Item Hangar", "Container", "Container", "Container"}},
		}
		for _, tc := range cases {
			got := makeNamesPath(ac, tc.item)
			assert.Equal(t, tc.want, got)
		}
	})
	t.Run("can return asset nodes by item IDs", func(t *testing.T) {
		cases := []struct {
			itemID int64
			found  bool
		}{
			{a1.ItemID, true},
			{a5.ItemID, true},
			{a2.ItemID, true},
			{a3.ItemID, true},
			{a4.ItemID, true},
			{b1.ItemID, true},
			{b2.ItemID, true},
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
	a := createCharacterAsset(assetParams{LocationID: locationID})
	b := createCharacterAsset(assetParams{LocationID: a.ItemID})
	c := createCharacterAsset(assetParams{LocationID: b.ItemID})
	d := createCharacterAsset(assetParams{LocationID: c.ItemID})
	e := createCharacterAsset(assetParams{LocationID: c.ItemID})
	f := createCharacterAsset(assetParams{LocationID: c.ItemID})
	assets := []*app.CharacterAsset{a, b, c, d, e, f}
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	ac := asset.NewFromCharacterAssets(assets, locations)
	// ac.MustNode()
	t.Run("can walk branch", func(t *testing.T) {
		an, _ := ac.Node(a.ItemID)
		s := slices.Collect(an.All())
		got := slices.Collect(xiter.MapSlice(s, func(x *asset.Node) int64 {
			return x.ID()
		}))
		want := slices.Collect(xiter.MapSlice(assets, func(x *app.CharacterAsset) int64 {
			return x.ID()
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
	ship1 := createCharacterAsset(assetParams{
		ItemID:     1,
		LocationID: locationID,
		Type:       shipType(),
	})
	drone := createCharacterAsset(assetParams{
		ItemID:       2,
		LocationID:   ship1.ItemID,
		LocationFlag: app.FlagDroneBay,
		Type: &app.EveType{
			Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryDrone}},
			Name:  "Hobgoblin",
		},
	})
	mineral1 := createCharacterAsset(assetParams{
		ItemID:     3,
		LocationID: locationID,
		Type: &app.EveType{
			Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryMineral}},
			Name:  "Tritanium",
		},
	})
	ship2 := createCharacterAsset(assetParams{
		ItemID:     4,
		LocationID: locationID,
		Type: &app.EveType{
			Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}},
			Name:  "Tristan",
		},
	})
	assets := []*app.CharacterAsset{ship1, mineral1, drone, ship2}
	loc := &app.EveLocation{ID: locationID, Name: "Alpha"}
	locations := []*app.EveLocation{loc}
	ac := asset.NewFromCharacterAssets(assets, locations)
	tree := mustLocation(ac, locationID)

	assert.Equal(t, 2, tree.ChildrenCount())
	assert.Equal(t, []string{"Alpha", "Item Hangar"}, makeNamesPath(ac, mineral1))
	assert.Equal(t, []string{"Alpha", "Ship Hangar", "Merlin", "Drone Bay"}, makeNamesPath(ac, drone))
	assert.Equal(t, []string{"Alpha", "Ship Hangar"}, makeNamesPath(ac, ship2))
	printTrees(ac)
	// t.Fail()
}

func TestCollection_Impounded(t *testing.T) {
	const locationID = 60007927
	office := createCorporationAsset(assetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   locationID,
		LocationFlag: app.FlagImpounded,
		LocationType: app.TypeStation,
		Type:         officeType(),
	})
	item1 := createCorporationAsset(assetParams{
		Quantity:     99,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG1,
		LocationType: app.TypeItem,
		Type:         tritaniumType(),
	})
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	assets := []*app.CorporationAsset{office, item1}
	ac := asset.NewFromCorporationAssets(assets, locations)

	assert.Equal(t, []string{"Alpha", "Impounded", "Office", "1st Division"}, makeNamesPath(ac, item1))

	printTrees(ac)
	// t.Fail()
}

func TestCollection_Offices(t *testing.T) {
	const locationID = 60007927
	office := createCorporationAsset(assetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   locationID,
		LocationFlag: app.FlagOfficeFolder,
		LocationType: app.TypeStation,
		Type:         officeType(),
	})
	item1 := createCorporationAsset(assetParams{
		Quantity:     99,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG1,
		LocationType: app.TypeItem,
		Type:         tritaniumType(),
	})
	item2 := createCorporationAsset(assetParams{
		Quantity:     33,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG2,
		LocationType: app.TypeItem,
		Type:         tritaniumType(),
	})
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	assets := []*app.CorporationAsset{office, item1, item2}

	ac := asset.NewFromCorporationAssets(assets, locations)

	assert.Equal(t, []string{"Alpha", "Office", "1st Division"}, makeNamesPath(ac, item1))

	officeNode := mustNode(ac, office.ItemID)
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
	printTrees(ac)
	// t.Fail()
}

func TestCollection_FilterAndItemCounts_Character(t *testing.T) {
	const (
		alphaID   = 60000001
		bravoID   = 30000001
		charlieID = 60000002
	)
	item1 := createCharacterAsset(assetParams{
		Quantity:   99,
		LocationID: alphaID,
	})
	item2 := createCharacterAsset(assetParams{
		Quantity:   3,
		LocationID: alphaID,
		Type:       cargoContainerType(),
	})
	item3 := createCharacterAsset(assetParams{
		Quantity:    1,
		IsSingleton: true,
		LocationID:  item2.ItemID,
	})
	ship1 := createCharacterAsset(assetParams{
		Quantity:   5,
		LocationID: alphaID,
		Type:       shipType(),
	})
	deliveryItem1 := createCharacterAsset(assetParams{
		Quantity:     42,
		LocationID:   alphaID,
		LocationFlag: app.FlagCapsuleerDeliveries,
		LocationType: app.TypeStation,
	})
	deliveryItem2 := createCharacterAsset(assetParams{
		Quantity:     4,
		LocationID:   alphaID,
		LocationFlag: app.FlagCapsuleerDeliveries,
		LocationType: app.TypeStation,
	})
	safetyWrap1 := createCharacterAsset(assetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   charlieID,
		LocationFlag: app.FlagAssetSafety,
		LocationType: app.TypeStation,
		Type:         assetSafetyWrapType(),
	})
	safetyItem1 := createCharacterAsset(assetParams{
		Quantity:   42,
		LocationID: safetyWrap1.ItemID,
	})
	spaceItem1 := createCharacterAsset(assetParams{
		Name:         "Anna",
		IsSingleton:  true,
		LocationFlag: app.FlagAutoFit,
		LocationType: app.TypeSolarSystem,
		LocationID:   bravoID,
		Quantity:     1,
		Type:         customsOfficeType(),
	})
	spaceItem2 := createCharacterAsset(assetParams{
		Name:         "Bob",
		IsSingleton:  true,
		LocationFlag: app.FlagAutoFit,
		LocationType: app.TypeSolarSystem,
		LocationID:   bravoID,
		Quantity:     1,
		Type:         customsOfficeType(),
	})
	locations := []*app.EveLocation{
		{
			ID:   alphaID,
			Name: "Alpha",
		},
		{
			ID:   bravoID,
			Name: "Bravo",
		},
		{
			ID:   charlieID,
			Name: "Charlie",
		},
	}
	assets := []*app.CharacterAsset{
		deliveryItem1,
		deliveryItem2,
		item1,
		item2,
		item3,
		ship1,
		safetyItem1,
		safetyWrap1,
		spaceItem1,
		spaceItem2,
	}

	t.Run("no filter", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		ac.ApplyFilter(asset.FilterNone)

		assert.Len(t, ac.Trees(), 3)

		alpha := mustLocation(ac, alphaID)
		assert.Equal(t, 3, alpha.ChildrenCount())
		assert.Equal(t, []string{"Alpha", "Item Hangar"}, makeNamesPath(ac, item1))
		assert.Equal(t, []string{"Alpha", "Deliveries"}, makeNamesPath(ac, deliveryItem1))

		bravo := mustLocation(ac, bravoID)
		assert.Equal(t, 1, bravo.ChildrenCount())
		assert.Equal(t, []string{"Bravo", "In Space"}, makeNamesPath(ac, spaceItem1))

		delta := mustLocation(ac, charlieID)
		assert.Equal(t, 1, delta.ChildrenCount())
		assert.Equal(t, []string{"Charlie", "Asset Safety", "Asset Safety Wrap"}, makeNamesPath(ac, safetyItem1))

		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("deliveries filter", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		ac.ApplyFilter(asset.FilterDeliveries)

		assert.Len(t, ac.Trees(), 1)
		assert.Equal(t, 1, mustLocation(ac, alphaID).ChildrenCount())
		assert.Equal(t, []string{"Alpha", "Deliveries"}, makeNamesPath(ac, deliveryItem1))

		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("personal assets filter", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		ac.ApplyFilter(asset.FilterPersonalAssets)

		assert.Len(t, ac.Trees(), 1)
		assert.Equal(t, []string{"Alpha", "Item Hangar"}, makeNamesPath(ac, item1))
		assert.Equal(t, []string{"Alpha", "Item Hangar"}, makeNamesPath(ac, item2))
		assert.Equal(t, []string{"Alpha", "Item Hangar", "Container"}, makeNamesPath(ac, item3))

		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("safety filter", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		ac.ApplyFilter(asset.FilterSafety)

		assert.Len(t, ac.Trees(), 1)
		assert.Equal(t, []string{"Charlie", "Asset Safety", "Asset Safety Wrap"}, makeNamesPath(ac, safetyItem1))

		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("item counts", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		ac.UpdateItemCounts()

		xassert.Equal(t, []int{5, 2}, makeCountsPath(ac, item1))
		xassert.Equal(t, []int{5, 1}, makeCountsPath(ac, ship1))
		xassert.Equal(t, []int{5, 2}, makeCountsPath(ac, deliveryItem1))
		xassert.Equal(t, []int{2, 2}, makeCountsPath(ac, spaceItem1))
		xassert.Equal(t, []int{1, 1, 1}, makeCountsPath(ac, safetyItem1))
		printTrees(ac)
		// assert.Fail(t, "STOP")

		// xassert.Equal(t, []int{2, 2, 2, 2}, makeCountsPath(ac, safetyItem2))
		// echo := mustLocation(ac, echoID)
		// printTrees(ac)
		// assert.Fail(t, "STOP")
	})
}

func TestCollection_FilterAndItemCounts_Corporation(t *testing.T) {
	const (
		alphaID   = 60000001
		bravoID   = 30000001
		charlieID = 60000002
		deltaID   = 60000003
		echoID    = 60000004
	)
	office1 := createCorporationAsset(assetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   alphaID,
		LocationFlag: app.FlagOfficeFolder,
		LocationType: app.TypeStation,
		Type:         officeType(),
	})
	officeItem1 := createCorporationAsset(assetParams{
		Quantity:     99,
		LocationID:   office1.ItemID,
		LocationFlag: app.FlagCorpSAG1,
	})
	officeItem2 := createCorporationAsset(assetParams{
		Quantity:     3,
		LocationID:   office1.ItemID,
		LocationFlag: app.FlagCorpSAG1,
	})
	officeItem3 := createCorporationAsset(assetParams{
		Quantity:     5,
		LocationID:   office1.ItemID,
		LocationFlag: app.FlagCorpSAG2,
	})
	impounded := createCorporationAsset(assetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   charlieID,
		LocationFlag: app.FlagImpounded,
		LocationType: app.TypeStation,
		Type:         officeType(),
	})
	impoundedItem1 := createCorporationAsset(assetParams{
		Quantity:     99,
		LocationID:   impounded.ItemID,
		LocationFlag: app.FlagCorpSAG1,
	})
	impoundedItem2 := createCorporationAsset(assetParams{
		Quantity:     5,
		LocationID:   impounded.ItemID,
		LocationFlag: app.FlagCorpSAG2,
	})
	impoundedItem3 := createCorporationAsset(assetParams{
		Quantity:     7,
		LocationID:   impounded.ItemID,
		LocationFlag: app.FlagCorpSAG2,
	})
	impoundedItem4 := createCorporationAsset(assetParams{
		Quantity:     7,
		LocationID:   impounded.ItemID,
		LocationFlag: app.FlagCorpSAG2,
	})
	deliveryItem1 := createCorporationAsset(assetParams{
		Quantity:     42,
		LocationID:   alphaID,
		LocationFlag: app.FlagCapsuleerDeliveries,
		LocationType: app.TypeStation,
	})
	deliveryItem2 := createCorporationAsset(assetParams{
		Quantity:     4,
		LocationID:   alphaID,
		LocationFlag: app.FlagCapsuleerDeliveries,
		LocationType: app.TypeStation,
		Type:         shipType(),
	})
	spaceItem1 := createCorporationAsset(assetParams{
		Name:         "Anna",
		IsSingleton:  true,
		LocationFlag: app.FlagAutoFit,
		LocationType: app.TypeSolarSystem,
		LocationID:   bravoID,
		Quantity:     1,
		Type:         customsOfficeType(),
	})
	spaceItem2 := createCorporationAsset(assetParams{
		Name:         "Bob",
		IsSingleton:  true,
		LocationFlag: app.FlagAutoFit,
		LocationType: app.TypeSolarSystem,
		LocationID:   bravoID,
		Quantity:     1,
		Type:         customsOfficeType(),
	})
	safetyWrap2 := createCorporationAsset(assetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   echoID,
		LocationFlag: app.FlagAssetSafety,
		LocationType: app.TypeStation,
		Type:         assetSafetyWrapType(),
	})
	safetyItem2 := createCorporationAsset(assetParams{
		Quantity:     99,
		LocationID:   safetyWrap2.ItemID,
		LocationFlag: app.FlagCorpDeliveries,
	})
	safetyItem3 := createCorporationAsset(assetParams{
		Quantity:     5,
		LocationID:   safetyWrap2.ItemID,
		LocationFlag: app.FlagCorpDeliveries,
	})
	structureCargoItem := createCorporationAsset(assetParams{
		Quantity:     5,
		LocationID:   deltaID,
		LocationFlag: app.FlagCargo,
	})
	locations := []*app.EveLocation{
		{
			ID:   alphaID,
			Name: "Alpha",
		},
		{
			ID:   bravoID,
			Name: "Bravo",
		},
		{
			ID:   charlieID,
			Name: "Charlie",
		},
		{
			ID:   deltaID,
			Name: "Delta",
		},
		{
			ID:   echoID,
			Name: "Echo",
		},
	}
	assets := []*app.CorporationAsset{
		deliveryItem1,
		deliveryItem2,
		impounded,
		impoundedItem1,
		impoundedItem2,
		impoundedItem3,
		impoundedItem4,
		office1,
		officeItem1,
		officeItem2,
		officeItem3,
		safetyItem2,
		safetyItem2,
		safetyItem3,
		safetyWrap2,
		spaceItem1,
		spaceItem2,
		structureCargoItem,
	}

	t.Run("no filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)

		assert.Len(t, ac.Trees(), 5)

		assert.Equal(t, 2, mustLocation(ac, alphaID).ChildrenCount())
		assert.Equal(t, []string{"Alpha", "Office", "1st Division"}, makeNamesPath(ac, officeItem1))
		assert.Equal(t, []string{"Alpha", "Deliveries"}, makeNamesPath(ac, deliveryItem1))

		assert.Equal(t, 1, mustLocation(ac, bravoID).ChildrenCount())
		assert.Equal(t, []string{"Bravo", "In Space"}, makeNamesPath(ac, spaceItem1))

		assert.Equal(t, 1, mustLocation(ac, charlieID).ChildrenCount())
		assert.Equal(t, []string{"Charlie", "Impounded", "Office", "1st Division"}, makeNamesPath(ac, impoundedItem1))

		assert.Equal(t, 1, mustLocation(ac, deltaID).ChildrenCount())
		assert.Equal(t, []string{"Delta", "Cargo Bay"}, makeNamesPath(ac, structureCargoItem))

		assert.Equal(t, 1, mustLocation(ac, echoID).ChildrenCount())
		assert.Equal(t, []string{"Echo", "Asset Safety", "Asset Safety Wrap", "Deliveries"}, makeNamesPath(ac, safetyItem3))

		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("deliveries filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		ac.ApplyFilter(asset.FilterDeliveries)

		assert.Len(t, ac.Trees(), 1)
		alpha := mustLocation(ac, alphaID)
		assert.Equal(t, 1, alpha.ChildrenCount())

		assert.Equal(t, []string{"Alpha", "Deliveries"}, makeNamesPath(ac, deliveryItem1))

		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("impounded filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		ac.ApplyFilter(asset.FilterImpounded)

		assert.Len(t, ac.Trees(), 1)
		assert.Equal(t, []string{"Charlie", "Impounded", "Office", "1st Division"}, makeNamesPath(ac, impoundedItem1))

		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("office filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		ac.ApplyFilter(asset.FilterOffice)

		assert.Len(t, ac.Trees(), 1)
		assert.Equal(t, []string{"Alpha", "Office", "1st Division"}, makeNamesPath(ac, officeItem1))

		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("safety filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		ac.ApplyFilter(asset.FilterSafety)

		assert.Len(t, ac.Trees(), 1)
		assert.Equal(t, []string{"Echo", "Asset Safety", "Asset Safety Wrap", "Deliveries"}, makeNamesPath(ac, safetyItem2))

		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("other filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		ac.ApplyFilter(asset.FilterCorpOther)

		assert.Len(t, ac.Trees(), 1)
		delta := mustLocation(ac, deltaID)
		assert.Equal(t, 1, delta.ChildrenCount())
		assert.Equal(t, []string{"Delta", "Cargo Bay"}, makeNamesPath(ac, structureCargoItem))
		printTrees(ac)
		// assert.Fail(t, "STOP")
	})

	t.Run("item counts", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		ac.UpdateItemCounts()

		xassert.Equal(t, []int{5, 3, 2}, makeCountsPath(ac, officeItem1))
		xassert.Equal(t, []int{5, 3, 1}, makeCountsPath(ac, officeItem3))
		xassert.Equal(t, []int{5, 2}, makeCountsPath(ac, deliveryItem1))
		xassert.Equal(t, []int{2, 2}, makeCountsPath(ac, spaceItem1))
		xassert.Equal(t, []int{4, 4, 4, 1}, makeCountsPath(ac, impoundedItem1))
		xassert.Equal(t, []int{1, 1}, makeCountsPath(ac, structureCargoItem))
		xassert.Equal(t, []int{2, 2, 2, 2}, makeCountsPath(ac, safetyItem2))

		mustLocation(ac, alphaID).PrintTree()
		mustLocation(ac, bravoID).PrintTree()
		mustLocation(ac, charlieID).PrintTree()
		mustLocation(ac, deltaID).PrintTree()
		mustLocation(ac, echoID).PrintTree()
		// assert.Fail(t, "STOP")
	})
}

// test helpers

var sequence atomic.Int64

func makeNamesPath(ac asset.Collection, it asset.Item) []string {
	return xslices.Map(mustNode(ac, it.ID()).Path(), func(x *asset.Node) string {
		return x.DisplayName()
	})
}

func makeCountsPath(ac asset.Collection, it asset.Item) []int {
	return xslices.Map(mustNode(ac, it.ID()).Path(), func(x *asset.Node) int {
		return x.ItemCount().ValueOrZero()
	})
}

func tritaniumType() *app.EveType {
	return &app.EveType{
		ID:    34,
		Group: &app.EveGroup{ID: 18, Category: &app.EveCategory{ID: app.EveCategoryMineral}},
		Name:  "Tritanium",
	}
}

func officeType() *app.EveType {
	return &app.EveType{
		ID:    27,
		Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryStation}},
		Name:  "Office",
	}
}

func cargoContainerType() *app.EveType {
	return &app.EveType{
		ID:    3293,
		Group: &app.EveGroup{ID: 12, Category: &app.EveCategory{ID: 2}},
		Name:  "Container",
	}
}

func shipType() *app.EveType {
	return &app.EveType{
		ID:    603,
		Group: &app.EveGroup{ID: 25, Category: &app.EveCategory{ID: app.EveCategoryShip}},
		Name:  "Merlin",
	}
}

func assetSafetyWrapType() *app.EveType {
	return &app.EveType{
		ID:    60,
		Group: &app.EveGroup{ID: 1319, Category: &app.EveCategory{ID: 29}},
		Name:  "Asset Safety Wrap",
	}
}

func customsOfficeType() *app.EveType {
	return &app.EveType{
		ID:    2233,
		Group: &app.EveGroup{ID: 1025, Category: &app.EveCategory{ID: 46}},
		Name:  "Customs Office",
	}
}

type assetParams struct {
	IsSingleton  bool
	ItemID       int64
	LocationFlag app.LocationFlag
	LocationID   int64
	LocationType app.LocationType
	Name         string
	Quantity     int
	Type         *app.EveType
}

func createCharacterAsset(arg assetParams) *app.CharacterAsset {
	return &app.CharacterAsset{
		Asset:       createAsset(arg),
		CharacterID: 1001,
	}
}

func createCorporationAsset(arg assetParams) *app.CorporationAsset {
	return &app.CorporationAsset{
		Asset:         createAsset(arg),
		CorporationID: 2001,
	}
}

func createAsset(arg assetParams) app.Asset {
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
		arg.Type = tritaniumType()
	}
	return app.Asset{
		IsSingleton:  arg.IsSingleton,
		ItemID:       arg.ItemID,
		LocationFlag: arg.LocationFlag,
		LocationID:   arg.LocationID,
		LocationType: arg.LocationType,
		Name:         arg.Name,
		Quantity:     arg.Quantity,
		Type:         arg.Type,
	}
}

func mustLocation(ac asset.Collection, locationID int64) *asset.Node {
	n, ok := ac.LocationTree(locationID)
	if !ok {
		panic("location not found")
	}
	return n
}

// mustNode returns the node for an ID or panics if not found.
func mustNode(ac asset.Collection, itemID int64) *asset.Node {
	n, ok := ac.Node(itemID)
	if !ok {
		panic(fmt.Sprintf("node not found for ID %d", itemID))
	}
	return n
}

func printTrees(ac asset.Collection) {
	trees := ac.Trees()
	slices.SortFunc(trees, func(a, b *asset.Node) int {
		return strings.Compare(a.DisplayName(), b.DisplayName())
	})
	for _, root := range trees {
		root.PrintTree()
	}
}
