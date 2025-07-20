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

func TestColonies_CanRenderEmpty(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	cases := []struct {
		name      string
		isDesktop bool
		filename  string
		size      fyne.Size
	}{
		{"desktop", true, "desktop_empty", fyne.NewSize(1700, 300)},
		{"mobile", false, "mobile_empty", fyne.NewSize(500, 800)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			test.ApplyTheme(t, test.Theme())
			ui := NewFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
			w := test.NewWindow(ui.colonies)
			defer w.Close()
			w.Resize(tc.size)

			ui.colonies.update()

			test.AssertImageMatches(t, "colonies/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}

func TestColonies_CanRenderFull(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	er := factory.CreateEveRegion(storage.CreateEveRegionParams{Name: "Black Rise"})
	con := factory.CreateEveConstellation(storage.CreateEveConstellationParams{RegionID: er.ID})
	system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
		SecurityStatus:  0.3,
		ConstellationID: con.ID,
		Name:            "Abune",
	})
	planetType := factory.CreateEveType(storage.CreateEveTypeParams{
		ID:   app.EveTypePlanetTemperate,
		Name: "Planet (Temperate)",
	})
	planet := factory.CreateEvePlanet(storage.CreateEvePlanetParams{
		Name:          "Abune I",
		SolarSystemID: system.ID,
		TypeID:        planetType.ID,
	})

	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	character := factory.CreateCharacter(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	p := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
		CharacterID: character.ID,
		EvePlanetID: planet.ID,
	})
	extractorGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{
		ID: app.EveGroupExtractorControlUnits,
	})
	extractor := factory.CreateEveType(storage.CreateEveTypeParams{
		GroupID: extractorGroup.ID,
	})
	extracted := factory.CreateEveType(storage.CreateEveTypeParams{
		Name: "Alpha",
	})
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID:      p.ID,
		TypeID:                 extractor.ID,
		ExpiryTime:             time.Now().UTC().Add(-time.Hour),
		ExtractorProductTypeID: optional.New(extracted.ID),
	})
	processorGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{
		ID: app.EveGroupProcessors,
	})
	processor := factory.CreateEveType(storage.CreateEveTypeParams{
		GroupID: processorGroup.ID,
	})
	schematic := factory.CreateEveSchematic(storage.CreateEveSchematicParams{Name: "Bravo"})
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID: p.ID,
		TypeID:            processor.ID,
		SchematicID:       optional.New(schematic.ID),
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
			test.ApplyTheme(t, test.Theme())
			ui := NewFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
			w := test.NewWindow(ui.colonies)
			defer w.Close()
			w.Resize(tc.size)

			ui.colonies.update()

			test.AssertImageMatches(t, "colonies/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}
