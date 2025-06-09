package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestCharacterImplants_CanRenderWithData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	character := factory.CreateCharacter()
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
		Section:     app.SectionImplants,
	})
	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
	ui.setCharacter(character)
	x := ui.characterImplants
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))

	x.update()

	test.AssertImageMatches(t, "characterimplants/master.png", w.Canvas().Capture())
}
