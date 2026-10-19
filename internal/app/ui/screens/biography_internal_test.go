package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestBiography(t *testing.T) {
	t.Run("shows description when character is exchanged", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewBiography(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := &app.Character{
			ID:           42,
			EveCharacter: &app.EveCharacter{ID: 42, Description: optional.New("Hello there")},
		}
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		xassert.Equal(t, "Hello there", a.body.Text)
		xassert.Equal(t, widget.MediumImportance, a.body.Importance)
	})
	t.Run("shows waiting message when character is exchanged to none", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewBiography(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), nil)
		// then
		xassert.Equal(t, "Waiting for character data to be loaded...", a.body.Text)
		xassert.Equal(t, widget.WarningImportance, a.body.Importance)
	})
	t.Run("shows waiting message when character has no eve character data", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewBiography(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := &app.Character{ID: 42}
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		xassert.Equal(t, "Waiting for character data to be loaded...", a.body.Text)
		xassert.Equal(t, widget.WarningImportance, a.body.Importance)
	})
	t.Run("updates when eve character of current character changed", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewBiography(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.character.Store(&app.Character{
			ID:           42,
			EveCharacter: &app.EveCharacter{ID: 42, Description: optional.New("Hello there")},
		})
		// when
		a.u.Signals().EveUniverseSectionChanged.Emit(t.Context(), app.EveUniverseSectionUpdated{
			Section: app.SectionEveCharacters,
			Changed: set.Of[int64](42),
		})
		// then
		xassert.Equal(t, "Hello there", a.body.Text)
	})
	t.Run("ignores change of other eve character", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewBiography(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.character.Store(&app.Character{
			ID:           42,
			EveCharacter: &app.EveCharacter{ID: 42, Description: optional.New("Hello there")},
		})
		// when
		a.u.Signals().EveUniverseSectionChanged.Emit(t.Context(), app.EveUniverseSectionUpdated{
			Section: app.SectionEveCharacters,
			Changed: set.Of[int64](7),
		})
		// then
		assert.Empty(t, a.body.Text)
	})
	t.Run("ignores change of other section", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewBiography(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.character.Store(&app.Character{
			ID:           42,
			EveCharacter: &app.EveCharacter{ID: 42, Description: optional.New("Hello there")},
		})
		// when
		a.u.Signals().EveUniverseSectionChanged.Emit(t.Context(), app.EveUniverseSectionUpdated{
			Section: app.SectionEveCorporations,
			Changed: set.Of[int64](42),
		})
		// then
		assert.Empty(t, a.body.Text)
	})
	t.Run("ignores section change without current character", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewBiography(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		// when
		a.u.Signals().EveUniverseSectionChanged.Emit(t.Context(), app.EveUniverseSectionUpdated{
			Section: app.SectionEveCharacters,
			Changed: set.Of[int64](42),
		})
		// then
		assert.Empty(t, a.body.Text)
	})
}
