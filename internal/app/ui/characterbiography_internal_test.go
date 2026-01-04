package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestCharacterBiography_CanRenderWithData(t *testing.T) {
	if testing.Short() {
		t.Skip(SkipUIReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Description: "This is a description",
	})
	character := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: ec.ID})
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	ui.setCharacter(character)
	w := test.NewWindow(ui.characterBiography)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))

	ui.characterBiography.update()

	test.AssertImageMatches(t, "characterbiography/master.png", w.Canvas().Capture())
}
