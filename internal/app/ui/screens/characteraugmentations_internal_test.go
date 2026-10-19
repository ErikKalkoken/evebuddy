package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterAugmentations(t *testing.T) {
	t.Run("shows implants of exchanged character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		i1 := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		i2 := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		xassert.Equal(t, "2 implants", a.top.Text)
		xassert.Equal(t, widget.MediumImportance, a.top.Importance)
		got := xslices.Map(a.implants, func(x *app.CharacterImplant) int64 {
			return x.EveType.ID
		})
		assert.ElementsMatch(t, []int64{i1.EveType.ID, i2.EveType.ID}, got)
	})
	t.Run("shows no character when character is exchanged to none", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.implants = []*app.CharacterImplant{factory.CreateCharacterImplant()}
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), nil)
		// then
		xassert.Equal(t, "No character", a.top.Text)
		xassert.Equal(t, widget.LowImportance, a.top.Importance)
		assert.Empty(t, a.implants)
	})
	t.Run("shows no data when section was never updated", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		a.implants = []*app.CharacterImplant{factory.CreateCharacterImplant()}
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		xassert.Equal(t, "No data", a.top.Text)
		xassert.Equal(t, widget.WarningImportance, a.top.Importance)
		assert.Empty(t, a.implants)
	})
	t.Run("shows error when section status can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		if _, err := db.Exec("DROP TABLE character_section_status"); err != nil {
			t.Fatal(err)
		}
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		assert.Contains(t, a.top.Text, "ERROR:")
		xassert.Equal(t, widget.DangerImportance, a.top.Importance)
		assert.Empty(t, a.implants)
	})
	t.Run("shows error when implants can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		if _, err := db.Exec("DROP TABLE character_implants"); err != nil {
			t.Fatal(err)
		}
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		assert.Contains(t, a.top.Text, "ERROR:")
		xassert.Equal(t, widget.DangerImportance, a.top.Importance)
		assert.Empty(t, a.implants)
	})
	t.Run("updates when implants of current character changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		// then
		xassert.Equal(t, "1 implant", a.top.Text)
		assert.Len(t, a.implants, 1)
	})
	t.Run("ignores change of other character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		a.character.Store(c)
		other := factory.CreateCharacterFull()
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: other.ID,
			Section:     app.SectionCharacterImplants,
		})
		// then
		assert.Empty(t, a.top.Text)
		assert.Empty(t, a.implants)
	})
	t.Run("ignores change of other section", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAttributes,
		})
		// then
		assert.Empty(t, a.top.Text)
		assert.Empty(t, a.implants)
	})
}
