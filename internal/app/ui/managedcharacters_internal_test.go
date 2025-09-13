package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestManagedCharacters_CanRenderWithData(t *testing.T) {
	if IsCI() {
		t.Skip("UI tests are currently flaky and therefore only run locally")
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	t.Run("normal", func(t *testing.T) {
		testutil.TruncateTables(db)
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			Name: "Bruce Wayne",
		})
		character := factory.CreateCharacter(storage.CreateCharacterParams{
			ID: ec.ID,
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: character.ID,
			Scopes:      app.Scopes(),
		})
		test.ApplyTheme(t, test.Theme())
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		ui.setCharacter(character)
		a := newManageCharacters(&manageCharactersWindow{
			u: ui,
		})
		w := test.NewWindow(a)
		defer w.Close()
		w.Resize(fyne.NewSize(600, 300))

		a.update()

		test.AssertImageMatches(t, "managedcharacters/master.png", w.Canvas().Capture())
	})
}
