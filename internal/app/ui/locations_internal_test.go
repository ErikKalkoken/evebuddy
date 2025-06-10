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
		SolarSystemID: optional.From(system.ID),
	})
	ship := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Merlin"})
	factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		LocationID: optional.From(location.ID),
		ID:         ec.ID,
		ShipID:     optional.From(ship.ID),
	})
	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
	x := ui.locations
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	x.update()

	test.AssertImageMatches(t, "locations/full.png", w.Canvas().Capture())
}

func TestLocations_CanRenderWithoutData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
	x := ui.locations
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	x.update()

	test.AssertImageMatches(t, "locations/minimal.png", w.Canvas().Capture())
}
