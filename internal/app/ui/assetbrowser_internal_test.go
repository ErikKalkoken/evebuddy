package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

// FIXME

// func TestCharacterAsset_CanRenderWithData(t *testing.T) {
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
// 	a := ui.characterAssets
// 	w := test.NewWindow(a)
// 	defer w.Close()
// 	w.Resize(fyne.NewSize(1700, 300))

// 	a.update()
// 	a.locations.OpenAllBranches()
// 	uid, ok := a.containerLocations[loc.ID]
// 	if !ok {
// 		t.Fail()
// 	}
// 	a.locations.Select(uid)

// 	test.AssertImageMatches(t, "characterasset/full.png", w.Canvas().Capture())
// }

// func TestCharacterAsset_CanRenderWithoutData(t *testing.T) {
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
// 	a := ui.characterAssets
// 	w := test.NewWindow(a)
// 	defer w.Close()
// 	w.Resize(fyne.NewSize(1700, 300))

// 	a.update()

// 	test.AssertImageMatches(t, "characterasset/minimal.png", w.Canvas().Capture())
// }
