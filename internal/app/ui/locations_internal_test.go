package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestLocations_CanRenderWithData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	er := factory.CreateEveRegion(storage.CreateEveRegionParams{Name: "Black Rise"})
	con := factory.CreateEveConstellation(storage.CreateEveConstellationParams{RegionID: er.ID})
	system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
		SecurityStatus:  0.3,
		ConstellationID: con.ID,
	})
	location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
		Name:          "Batcave",
		SolarSystemID: optional.New(system.ID),
	})
	ship := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Merlin"})
	factory.CreateCharacter(storage.CreateCharacterParams{
		LocationID: optional.New(location.ID),
		ID:         ec.ID,
		ShipID:     optional.New(ship.ID),
	})
	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
	w := test.NewWindow(ui.locations)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	ui.locations.update()

	test.AssertImageMatches(t, "locations/full.png", w.Canvas().Capture())
}

func TestLocations_CanRenderWithoutData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	factory.CreateCharacter(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
	w := test.NewWindow(ui.locations)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	ui.locations.update()

	test.AssertImageMatches(t, "locations/minimal.png", w.Canvas().Capture())
}
