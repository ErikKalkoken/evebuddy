package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestCharacterAssetsMakeLocationTreeData(t *testing.T) {
	t.Run("can create simple tree", func(t *testing.T) {
		el := &app.EveLocation{
			ID:          100000,
			Name:        "Alpha 1",
			SolarSystem: &app.EveSolarSystem{Name: "Alpha", ID: 1},
		}
		a := &app.CharacterAsset{ItemID: 1, LocationID: el.ID}
		b := &app.CharacterAsset{ItemID: 2, LocationID: 1}
		c := &app.CharacterAsset{ItemID: 3, LocationID: 2}
		d := &app.CharacterAsset{ItemID: 4, LocationID: 2}
		assets := []*app.CharacterAsset{a, b, c, d}
		locations := []*app.EveLocation{el}
		ac := assetcollection.New(assets, locations)
		tree := makeLocationTreeData(ac, 42)
		assert.False(t, tree.IsEmpty())
	})
	t.Run("can have multiple locations with items in space", func(t *testing.T) {
		l1 := &app.EveLocation{
			ID:          100000,
			Name:        "Alpha 1",
			SolarSystem: &app.EveSolarSystem{Name: "Alpha", ID: 1},
		}
		l2 := &app.EveLocation{
			ID:          100001,
			Name:        "Alpha 2",
			SolarSystem: &app.EveSolarSystem{Name: "Alpha", ID: 1},
		}
		a := &app.CharacterAsset{ItemID: 1, LocationID: l1.ID}
		b := &app.CharacterAsset{ItemID: 2, LocationID: 1}
		c := &app.CharacterAsset{ItemID: 3, LocationID: 1}
		d := &app.CharacterAsset{ItemID: 4, LocationID: 1}
		e := &app.CharacterAsset{ItemID: 5, LocationID: l2.ID}
		assets := []*app.CharacterAsset{a, b, c, d, e}
		locations := []*app.EveLocation{l1, l2}
		ac := assetcollection.New(assets, locations)
		tree := makeLocationTreeData(ac, 42)
		assert.False(t, tree.IsEmpty())
	})
}

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

func TestCharacterAsset_CanRenderWithData(t *testing.T) {
	if IsCI() {
		t.Skip("UI tests are currently flaky and therefore only run locally")
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	character := factory.CreateCharacterFull(storage.CreateCharacterParams{
		AssetValue: optional.New(1000000000.0),
	})
	et := factory.CreateEveType(storage.CreateEveTypeParams{
		ID:   42,
		Name: "Merlin",
	})
	system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
		ID:             1001,
		SecurityStatus: 0.2,
	})
	loc := factory.CreateEveLocationStation(storage.UpdateOrCreateLocationParams{
		Name:          "Abune - My castle",
		SolarSystemID: optional.New(system.ID),
	})
	factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
		CharacterID:  character.ID,
		EveTypeID:    et.ID,
		Quantity:     10,
		LocationID:   loc.ID,
		LocationType: "other",
		LocationFlag: app.FlagHangar,
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionCharacterAssets,
	})
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	ui.setCharacter(character)
	a := ui.characterAsset
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	a.update()
	a.locations.OpenAllBranches()
	uid, ok := a.containerLocations[loc.ID]
	if !ok {
		t.Fail()
	}
	a.locations.Select(uid)

	test.AssertImageMatches(t, "characterasset/full.png", w.Canvas().Capture())
}

func TestCharacterAsset_CanRenderWithoutData(t *testing.T) {
	if IsCI() {
		t.Skip("UI tests are currently flaky and therefore only run locally")
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	character := factory.CreateCharacter(storage.CreateCharacterParams{
		AssetValue: optional.New(1000000000.0),
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionCharacterAssets,
	})
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	ui.setCharacter(character)
	a := ui.characterAsset
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	a.update()

	test.AssertImageMatches(t, "characterasset/minimal.png", w.Canvas().Capture())
}
