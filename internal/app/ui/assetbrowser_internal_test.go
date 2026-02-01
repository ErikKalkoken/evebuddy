package ui

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/asset"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestSplitLines(t *testing.T) {
	const maxLine = 10
	cases := []struct {
		name  string
		in    string
		want1 string
		want2 string
	}{
		{"single line 1", "alpha", "alpha", ""},
		{"single line 2", "alpha boy", "alpha boy", ""},
		{"two lines single word", "verySophisticated", "verySophis", "ticated"},
		{"two lines long word", "verySophisticatedIndeed", "verySophis", "ticatedInd"},
		{"two lines", "first second", "first", "second"},
		{"two lines with truncation", "first second third", "first", "second thi"},
		{"one long word", "firstSecondThirdForth", "firstSecon", "dThirdFort"},
		{"special 1", "Erik Kalkoken's Cald", "Erik", "Kalkoken's"},
		// {"two lines two words", "Contaminated Nanite", "Contaminat", "ed Nanite"}, FIXME!
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got1, got2 := splitLines(tc.in, maxLine)
			assert.Equal(t, tc.want1, got1)
			assert.Equal(t, tc.want2, got2)
		})
	}
}

// func TestAssetBrowser_CanRenderWithData(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip(SkipUIReason)
// 	}
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()
// 	character := factory.CreateCharacterFull(storage.CreateCharacterParams{
// 		AssetValue: optional.New(1000000000.0),
// 	})
// 	et := factory.CreateEveType(storage.CreateEveTypeParams{
// 		ID:   42,
// 		Name: "Merlin",
// 	})
// 	system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
// 		ID:             1001,
// 		SecurityStatus: 0.2,
// 	})
// 	loc := factory.CreateEveLocationStation(storage.UpdateOrCreateLocationParams{
// 		Name:          "Abune - My castle",
// 		SolarSystemID: optional.New(system.ID),
// 	})
// 	factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
// 		CharacterID:  character.ID,
// 		EveTypeID:    et.ID,
// 		Quantity:     10,
// 		LocationID:   loc.ID,
// 		LocationType: app.TypeOther,
// 		LocationFlag: app.FlagHangar,
// 	})
// 	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
// 		CharacterID: character.ID,
// 		Section:     app.SectionCharacterAssets,
// 	})
// 	test.ApplyTheme(t, test.Theme())
// 	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
// 	ui.setCharacter(character)
// 	a := ui.characterAssetBrowser
// 	w := test.NewWindow(a)
// 	defer w.Close()
// 	w.Resize(fyne.NewSize(1700, 300))

// 	a.update()
// 	a.Navigation.navigation.OpenAllBranches()
// 	// n, ok := a.Navigation.ac.LocationTree(loc.ID)
// 	// require.True(t, ok)
// 	// a.Navigation.navigation.SelectNode(n) // FIXME
// 	test.AssertImageMatches(t, "characterassetbrowser/full.png", w.Canvas().Capture())
// }

// func TestAssetBrowser_CanRenderWithoutData(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip(SkipUIReason)
// 	}
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()
// 	character := factory.CreateCharacter(storage.CreateCharacterParams{
// 		AssetValue: optional.New(1000000000.0),
// 	})
// 	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
// 		CharacterID: character.ID,
// 		Section:     app.SectionCharacterAssets,
// 	})
// 	test.ApplyTheme(t, test.Theme())
// 	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
// 	ui.setCharacter(character)
// 	a := ui.characterAssetBrowser
// 	w := test.NewWindow(a)
// 	defer w.Close()
// 	w.Resize(fyne.NewSize(1700, 300))

// 	a.update()

// 	test.AssertImageMatches(t, "characterassetbrowser/minimal.png", w.Canvas().Capture())
// }

func TestGenerateTreeData_Character(t *testing.T) {
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
		IsSingleton: true,
		Quantity:    1,
		LocationID:  alphaID,
		Type:        cargoContainerType(),
	})
	item3 := createCharacterAsset(assetParams{
		IsSingleton: true,
		Quantity:    1,
		LocationID:  item2.ItemID,
	})
	ship1 := createCharacterAsset(assetParams{
		IsSingleton: true,
		Quantity:    1,
		LocationID:  alphaID,
		Type:        shipType(),
	})
	drone := createCharacterAsset(assetParams{
		Quantity:     1,
		IsSingleton:  true,
		LocationID:   ship1.ItemID,
		LocationFlag: app.FlagDroneBay,
		Type:         droneType(),
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
		drone,
		item1,
		item2,
		item3,
		safetyItem1,
		safetyWrap1,
		ship1,
		spaceItem1,
		spaceItem2,
	}

	t.Run("correct structure without filters", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetNoFilter, false)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Alpha", "Deliveries"},
			{"Alpha", "Item Hangar", "Container"},
			{"Alpha", "Ship Hangar", "Merlin", "Drone Bay"},
			{"Bravo", "In Space"},
			{"Charlie", "Asset Safety", "Asset Safety Wrap"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("deliveries filter", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetDeliveries, false)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Alpha", "Deliveries"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("personal assets filter", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetPersonalAssets, false)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Alpha", "Item Hangar", "Container"},
			{"Alpha", "Ship Hangar", "Merlin", "Drone Bay"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("safety filter", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetSafety, false)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Charlie", "Asset Safety", "Asset Safety Wrap"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("item counts", func(t *testing.T) {
		ac := asset.NewFromCharacterAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetNoFilter, false)

		xassert.Equal(t, []int{5, 2}, makeCountsPath(ac, td, item1))
		xassert.Equal(t, []int{5, 1}, makeCountsPath(ac, td, ship1))
		xassert.Equal(t, []int{5, 1, 0, 1}, makeCountsPath(ac, td, drone))
		xassert.Equal(t, []int{5, 2}, makeCountsPath(ac, td, deliveryItem1))
		xassert.Equal(t, []int{2, 2}, makeCountsPath(ac, td, spaceItem1))
		xassert.Equal(t, []int{1, 1, 1}, makeCountsPath(ac, td, safetyItem1))
		td.Print(nil)
		// assert.Fail(t, "STOP")
	})
}

func TestGenerateTreeData_Corporation(t *testing.T) {
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
		td := generateTreeData(ac.Locations(), assetNoFilter, true)

		got1 := xslices.Map(td.Children(nil), func(x *assetContainerNode) string {
			return x.String()
		})
		want1 := []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo"}
		assert.ElementsMatch(t, want1, got1)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Alpha", "Deliveries"},
			{"Alpha", "Office", "1st Division"},
			{"Alpha", "Office", "2nd Division"},
			{"Alpha", "Office", "3rd Division"},
			{"Alpha", "Office", "4th Division"},
			{"Alpha", "Office", "5th Division"},
			{"Alpha", "Office", "6th Division"},
			{"Alpha", "Office", "7th Division"},
			{"Bravo", "In Space"},
			{"Charlie", "Impounded", "Office", "1st Division"},
			{"Charlie", "Impounded", "Office", "2nd Division"},
			{"Charlie", "Impounded", "Office", "3rd Division"},
			{"Charlie", "Impounded", "Office", "4th Division"},
			{"Charlie", "Impounded", "Office", "5th Division"},
			{"Charlie", "Impounded", "Office", "6th Division"},
			{"Charlie", "Impounded", "Office", "7th Division"},
			{"Delta", "Cargo Bay"},
			{"Echo", "Asset Safety", "Asset Safety Wrap", "Deliveries"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})
	t.Run("deliveries filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetDeliveries, true)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Alpha", "Deliveries"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("impounded filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetImpounded, true)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Charlie", "Impounded", "Office", "1st Division"},
			{"Charlie", "Impounded", "Office", "2nd Division"},
			{"Charlie", "Impounded", "Office", "3rd Division"},
			{"Charlie", "Impounded", "Office", "4th Division"},
			{"Charlie", "Impounded", "Office", "5th Division"},
			{"Charlie", "Impounded", "Office", "6th Division"},
			{"Charlie", "Impounded", "Office", "7th Division"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("office filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetOffice, true)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Alpha", "Office", "1st Division"},
			{"Alpha", "Office", "2nd Division"},
			{"Alpha", "Office", "3rd Division"},
			{"Alpha", "Office", "4th Division"},
			{"Alpha", "Office", "5th Division"},
			{"Alpha", "Office", "6th Division"},
			{"Alpha", "Office", "7th Division"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("safety filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetSafety, true)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Echo", "Asset Safety", "Asset Safety Wrap", "Deliveries"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("in space filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetInSpace, true)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Bravo", "In Space"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("other filter", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetCorpOther, true)

		got := td.AllPaths(nil)
		want := [][]string{
			{"Delta", "Cargo Bay"},
		}
		assert.ElementsMatch(t, want, got)

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})

	t.Run("item counts", func(t *testing.T) {
		ac := asset.NewFromCorporationAssets(assets, locations)
		td := generateTreeData(ac.Locations(), assetNoFilter, true)

		xassert.Equal(t, []int{5, 3, 2}, makeCountsPath(ac, td, officeItem1))
		xassert.Equal(t, []int{5, 3, 1}, makeCountsPath(ac, td, officeItem3))
		xassert.Equal(t, []int{5, 2}, makeCountsPath(ac, td, deliveryItem1))
		xassert.Equal(t, []int{2, 2}, makeCountsPath(ac, td, spaceItem1))
		xassert.Equal(t, []int{4, 4, 4, 1}, makeCountsPath(ac, td, impoundedItem1))
		xassert.Equal(t, []int{1, 1}, makeCountsPath(ac, td, structureCargoItem))
		xassert.Equal(t, []int{2, 2, 2, 2}, makeCountsPath(ac, td, safetyItem2))

		td.Print(nil)
		// assert.Fail(t, "STOP")
	})
}

var sequence atomic.Int64

func makeCountsPath(ac asset.Tree, td iwidget.TreeData[assetContainerNode], it asset.Item) []int {
	n, ok := ac.Node(it.ID())
	if !ok {
		return nil
	}
	x, ok := findContainer(td, n.Parent())
	if !ok {
		return nil
	}
	return xslices.Map(td.Path(nil, x), func(x *assetContainerNode) int {
		return x.itemCount.ValueOrZero()
	})
}

func findContainer(td iwidget.TreeData[assetContainerNode], node *asset.Node) (*assetContainerNode, bool) {
	var found *assetContainerNode
	td.Walk(nil, func(n *assetContainerNode) bool {
		if n.node == node {
			found = n
			return false
		}
		return true
	})
	if found == nil {
		return nil, false
	}
	return found, true
}

func mineralType() *app.EveType {
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

func droneType() *app.EveType {
	return &app.EveType{
		ID:    2454,
		Group: &app.EveGroup{ID: 100, Category: &app.EveCategory{ID: app.EveCategoryDrone}},
		Name:  "Hobgoblin I",
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
		arg.Type = mineralType()
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
