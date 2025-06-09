package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestCharacters_CanRenderWithData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	alliance := factory.CreateEveEntityAlliance(app.EveEntity{
		Name: "Wayne Inc.",
	})
	corporation := factory.CreateEveEntityCorporation(app.EveEntity{
		Name: "Wayne Technolgy",
	})
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		AllianceID:     alliance.ID,
		Birthday:       time.Now().Add(-24 * 365 * 3 * time.Hour),
		CorporationID:  corporation.ID,
		Name:           "Bruce Wayne",
		SecurityStatus: -10.0,
	})
	homeSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
		SecurityStatus: 0.3,
	})
	home := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
		Name:          "Batcave",
		SolarSystemID: optional.From(homeSystem.ID),
	})
	character := factory.CreateCharacter(storage.CreateCharacterParams{
		AssetValue:    optional.From(12_000_000_000.0),
		HomeID:        optional.From(home.ID),
		ID:            ec.ID,
		WalletBalance: optional.From(23_000_000.0),
		LastLoginAt:   optional.From(time.Now().Add(-24 * 7 * 2 * time.Hour)),
	})
	factory.CreateCharacterMail(storage.CreateCharacterMailParams{
		CharacterID: character.ID,
		IsRead:      false,
	})
	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
	x := ui.characters
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	x.update()

	test.AssertImageMatches(t, "characters/master.png", w.Canvas().Capture())
}
