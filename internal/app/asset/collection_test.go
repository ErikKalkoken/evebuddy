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

func makePath(ac asset.Collection, id asset.Item) []string {
	return xslices.Map(ac.MustNode(id.ID()).Path(), func(x *asset.Node) string {
		return x.DisplayName()
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
		arg.Type = tritaniumType()
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
	const (
		alphaID = 100000
		bravoID = 101000
	)
	a1 := createCharacterAsset(characterAssetParams{LocationID: alphaID, Type: cargoContainerType()})
	a2 := createCharacterAsset(characterAssetParams{LocationID: a1.ItemID, Type: cargoContainerType()})
	a3 := createCharacterAsset(characterAssetParams{LocationID: a2.ItemID, Type: cargoContainerType()})
	a4 := createCharacterAsset(characterAssetParams{LocationID: a3.ItemID, Quantity: 3})
	a5 := createCharacterAsset(characterAssetParams{LocationID: alphaID})
	b1 := createCharacterAsset(characterAssetParams{LocationID: bravoID, Type: cargoContainerType()})
	b2 := createCharacterAsset(characterAssetParams{LocationID: b1.ItemID})
	assets := []*app.CharacterAsset{a1, a5, a2, a3, b1, b2, a4}
	loc1 := &app.EveLocation{ID: alphaID, Name: "Alpha"}
	loc2 := &app.EveLocation{ID: bravoID, Name: "Bravo"}
	locations := []*app.EveLocation{loc1, loc2}
	ac := asset.NewFromCharacterAssets(assets, locations)

	t.Run("can create trees from character assets", func(t *testing.T) {
		assert.Len(t, ac.Trees(), 2)

		t1 := ac.MustLocationTree(loc1.ID)
		assert.Equal(t, []string{"Alpha", "Item Hangar", "Container", "Container", "Container"}, makePath(ac, a4))
		assert.Equal(t, []string{"Alpha", "Item Hangar"}, makePath(ac, a5))
		asset.PrintTree(t1)

		t2 := ac.MustLocationTree(loc2.ID)
		assert.Equal(t, []string{"Bravo", "Item Hangar", "Container"}, makePath(ac, b2))
		asset.PrintTree(t2)
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
			got := makePath(ac, tc.item)
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
		Type:         officeType(),
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

	assert.Equal(t, 3, len(tree.Children()))
	assert.Equal(t, []string{"Alpha", "Item Hangar"}, makePath(ac, mineral1))
	assert.Equal(t, []string{"Alpha", "Ship Hangar", "Merlin", "Drone Bay"}, makePath(ac, drone))
	assert.Equal(t, []string{"Alpha", "Ship Hangar"}, makePath(ac, ship2))
	assert.Equal(t, []string{"Alpha", "Office", "1st Division"}, makePath(ac, mineral2))
	asset.PrintTree(tree)
	// t.Fail()
}

func TestCollection_Impounded(t *testing.T) {
	const locationID = 60007927
	office := createCharacterAsset(characterAssetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   locationID,
		LocationFlag: app.FlagImpounded,
		LocationType: app.TypeStation,
		Type:         officeType(),
	})
	item1 := createCharacterAsset(characterAssetParams{
		Quantity:     99,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG1,
		LocationType: app.TypeItem,
		Type:         tritaniumType(),
	})
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	assets := []*app.CharacterAsset{office, item1}
	ac := asset.NewFromCharacterAssets(assets, locations)
	root := ac.Trees()[0]

	assert.Equal(t, []string{"Alpha", "Impounded", "Office", "1st Division"}, makePath(ac, item1))

	asset.PrintTree(root)
	// t.Fail()
}

func TestCollection_Offices(t *testing.T) {
	const locationID = 60007927
	office := createCharacterAsset(characterAssetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   locationID,
		LocationFlag: app.FlagOfficeFolder,
		LocationType: app.TypeStation,
		Type:         officeType(),
	})
	item1 := createCharacterAsset(characterAssetParams{
		Quantity:     99,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG1,
		LocationType: app.TypeItem,
		Type:         tritaniumType(),
	})
	item2 := createCharacterAsset(characterAssetParams{
		Quantity:     33,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG2,
		LocationType: app.TypeItem,
		Type:         tritaniumType(),
	})
	locations := []*app.EveLocation{{ID: locationID, Name: "Alpha"}}
	assets := []*app.CharacterAsset{office, item1, item2}

	ac := asset.NewFromCharacterAssets(assets, locations)

	root := ac.Trees()[0]
	assert.Equal(t, []string{"Alpha", "Office", "1st Division"}, makePath(ac, item1))

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
	// t.Fail()
}

func TestCollection_2(t *testing.T) {
	const (
		alphaID = 60007927
		bravoID = 30002537
	)
	office := createCharacterAsset(characterAssetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   alphaID,
		LocationFlag: app.FlagOfficeFolder,
		LocationType: app.TypeStation,
		Type:         officeType(),
	})
	officeItem1 := createCharacterAsset(characterAssetParams{
		Quantity:     99,
		LocationID:   office.ItemID,
		LocationFlag: app.FlagCorpSAG1,
		Type:         tritaniumType(),
	})
	impounded := createCharacterAsset(characterAssetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   alphaID,
		LocationFlag: app.FlagImpounded,
		LocationType: app.TypeStation,
		Type:         officeType(),
	})
	officeItem2 := createCharacterAsset(characterAssetParams{
		Quantity:     99,
		LocationID:   impounded.ItemID,
		LocationFlag: app.FlagCorpSAG1,
		Type:         tritaniumType(),
	})
	deliveryItem := createCharacterAsset(characterAssetParams{
		Quantity:     42,
		LocationID:   alphaID,
		LocationFlag: app.FlagCapsuleerDeliveries,
		LocationType: app.TypeStation,
		Type:         tritaniumType(),
	})
	safetyWrap := createCharacterAsset(characterAssetParams{
		IsSingleton:  true,
		Quantity:     1,
		LocationID:   alphaID,
		LocationFlag: app.FlagAssetSafety,
		LocationType: app.TypeStation,
		Type:         assetSafetyWrapType(),
	})
	safetyItem := createCharacterAsset(characterAssetParams{
		Quantity:   42,
		LocationID: safetyWrap.ItemID,
		Type:       tritaniumType(),
	})
	spaceItem := createCharacterAsset(characterAssetParams{
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
	}
	assets := []*app.CharacterAsset{
		deliveryItem,
		impounded,
		office,
		officeItem1,
		officeItem2,
		safetyItem,
		safetyWrap,
		spaceItem,
	}
	acAll := asset.NewFromCharacterAssets(assets, locations)

	t.Run("no filter", func(t *testing.T) {
		assert.Len(t, acAll.Trees(), 2)
		alpha, ok := acAll.LocationTree(alphaID)
		require.True(t, ok)
		assert.Len(t, alpha.Children(), 4)
		assert.Equal(t, []string{"Alpha", "Office", "1st Division"}, makePath(acAll, officeItem1))
		assert.Equal(t, []string{"Alpha", "Deliveries"}, makePath(acAll, deliveryItem))
		assert.Equal(t, []string{"Alpha", "Asset Safety", "Asset Safety Wrap"}, makePath(acAll, safetyItem))
		assert.Equal(t, []string{"Alpha", "Impounded", "Office", "1st Division"}, makePath(acAll, officeItem2))
		asset.PrintTree(alpha)

		bravo, ok := acAll.LocationTree(bravoID)
		require.True(t, ok)
		assert.Equal(t, []string{"Bravo", "In Space"}, makePath(acAll, spaceItem))
		asset.PrintTree(bravo)
		// t.Fail()
	})

}
