package skills

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

func TestCharacterAttributes_CanRenderWithData(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
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
		Section:     app.SectionCharacterAttributes,
	})
	test.ApplyTheme(t, test.Theme())
	a := NewAttributes(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))

	a.character.Store(character)
	a.update(t.Context())

	test.AssertImageMatches(t, "characterattributes/master.png", w.Canvas().Capture())
}
