package characters

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestCharacterBiography_CanRenderWithData(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Description: optional.New("This is a description"),
	})
	character := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: ec.ID})
	test.ApplyTheme(t, test.Theme())
	signals := app.NewSignals()
	a := NewBiography(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
		Signals: signals,
	}))
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))

	signals.CurrentCharacterExchanged.Emit(t.Context(), character)

	test.AssertImageMatches(t, "biography/master.png", w.Canvas().Capture())
}
