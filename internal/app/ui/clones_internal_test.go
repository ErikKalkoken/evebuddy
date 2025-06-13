package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestClones_CanRenderLocationWithoutSystem(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	character := factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID: ec.ID,
	})

	location := factory.CreateEveLocationEmptyStructure(storage.UpdateOrCreateLocationParams{
		Name: "A Structure",
	})
	factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
		CharacterID: character.ID,
		LocationID:  location.ID,
	})

	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
	w := test.NewWindow(ui.clones)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	ui.clones.update()

	test.AssertImageMatches(t, "clones/empty_location.png", w.Canvas().Capture())
}

func TestClones_CanRenderEmpty(t *testing.T) {
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
			w := test.NewWindow(ui.clones)
			defer w.Close()
			w.Resize(tc.size)

			ui.clones.update()

			test.AssertImageMatches(t, "clones/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}

func TestClones_CanRenderFull(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

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

	ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	character1 := factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID: ec1.ID,
	})
	factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
		CharacterID: character1.ID,
		LocationID:  location.ID,
	})

	ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Clark Kent",
	})
	character2 := factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID: ec2.ID,
	})
	i1 := factory.CreateEveType()
	i2 := factory.CreateEveType()
	factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
		CharacterID: character2.ID,
		LocationID:  location.ID,
		Implants:    []int32{i1.ID, i2.ID},
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
			w := test.NewWindow(ui.clones)
			defer w.Close()
			w.Resize(tc.size)

			ui.clones.update()

			test.AssertImageMatches(t, "clones/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}
