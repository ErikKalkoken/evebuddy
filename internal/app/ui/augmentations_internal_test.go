package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestAugmentations_CanRenderWithData(t *testing.T) {
	if IsCI() {
		t.Skip("UI tests are currently flaky and therefore only run locally")
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
		ID:   42,
	})
	character := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
	et := factory.CreateEveType(storage.CreateEveTypeParams{
		Name: "Dummy Implant",
	})
	da := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: app.EveDogmaAttributeImplantSlot})
	factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
		DogmaAttributeID: da.ID,
		EveTypeID:        et.ID,
		Value:            3,
	})
	factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{
		CharacterID: character.ID,
		EveTypeID:   et.ID,
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionCharacterImplants,
	})
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	a := ui.augmentations
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))

	a.update()
	a.tree.OpenAllBranches()

	test.AssertImageMatches(t, "augmentations/master.png", w.Canvas().Capture())
}
