package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestAssets_CanRenderWithData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	character := factory.CreateCharacterFull(storage.CreateCharacterParams{
		AssetValue: optional.New(1000000000.0),
		ID:         ec.ID,
	})
	eg := factory.CreateEveGroup(storage.CreateEveGroupParams{
		Name: "Frigate",
	})
	et := factory.CreateEveType(storage.CreateEveTypeParams{
		ID:      42,
		Name:    "Merlin",
		GroupID: eg.ID,
	})
	factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
		TypeID:       et.ID,
		AveragePrice: 120000.0,
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
		LocationType: app.TypeOther,
		LocationFlag: app.FlagHangar,
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionCharacterAssets,
	})
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	ui.setCharacter(character)
	w := test.NewWindow(ui.assets)
	defer w.Close()
	w.Resize(fyne.NewSize(1400, 300))

	ui.assets.update()

	test.AssertImageMatches(t, "assets/master.png", w.Canvas().Capture())
}
