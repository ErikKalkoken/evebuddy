package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestCharacterAttributes_CanRenderWithData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t.TempDir())
	defer db.Close()
	character := factory.CreateCharacter()
	factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{
		CharacterID:  character.ID,
		Charisma:     21,
		Intelligence: 22,
		Memory:       23,
		Perception:   24,
		Willpower:    25,
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionAttributes,
	})
	ui := NewFakeBaseUI(st, test.NewTempApp(t))
	ui.setCharacter(character)
	x := ui.characterAttributes
	w := test.NewWindow(x)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))

	x.update()

	test.AssertImageMatches(t, "characterattributes/master.png", w.Canvas().Capture())
}
