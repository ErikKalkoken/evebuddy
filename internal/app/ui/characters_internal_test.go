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
	test.ApplyTheme(t, test.Theme())
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
		SolarSystemID: optional.New(homeSystem.ID),
	})
	character := factory.CreateCharacterFull(storage.CreateCharacterParams{
		AssetValue:    optional.New(12_000_000_000.0),
		HomeID:        optional.New(home.ID),
		ID:            ec.ID,
		WalletBalance: optional.New(23_000_000.0),
		LastLoginAt:   optional.New(time.Now().Add(-24 * 7 * 2 * time.Hour)),
	})
	factory.CreateCharacterMail(storage.CreateCharacterMailParams{
		CharacterID: character.ID,
		IsRead:      false,
	})

	cases := []struct {
		name      string
		isDesktop bool
		filename  string
		size      fyne.Size
	}{
		{"desktop", true, "desktop_full", fyne.NewSize(1700, 300)},
		{"mobile", false, "mobile_full", fyne.NewSize(500, 800)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ui := NewFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
			w := test.NewWindow(ui.characters)
			defer w.Close()
			w.Resize(tc.size)

			ui.characters.update()

			test.AssertImageMatches(t, "characters/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}

func TestCharacters_CanRenderWitoutData(t *testing.T) {
	test.ApplyTheme(t, test.Theme())
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	corporation := factory.CreateEveEntityCorporation(app.EveEntity{
		Name: "Wayne Technolgy",
	})
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Birthday:       time.Now().Add(-24 * 365 * 3 * time.Hour),
		CorporationID:  corporation.ID,
		Name:           "Bruce Wayne",
		SecurityStatus: -10.0,
	})
	factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID: ec.ID,
	})

	cases := []struct {
		name      string
		isDesktop bool
		filename  string
		size      fyne.Size
	}{
		{"desktop", true, "desktop_minimal", fyne.NewSize(1700, 300)},
		{"mobile", false, "mobile_minimal", fyne.NewSize(500, 800)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ui := NewFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
			w := test.NewWindow(ui.characters)
			defer w.Close()
			w.Resize(tc.size)

			ui.characters.update()

			test.AssertImageMatches(t, "characters/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}
