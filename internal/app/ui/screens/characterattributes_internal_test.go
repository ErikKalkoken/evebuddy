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
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterAttributeRow_IsComment(t *testing.T) {
	t.Run("row without points is a comment", func(t *testing.T) {
		// given
		r := characterAttributeRow{name: "Bonus Remaps Available: 1"}
		// when
		got := r.isComment()
		// then
		assert.True(t, got)
	})
	t.Run("row with points is not a comment", func(t *testing.T) {
		// given
		r := characterAttributeRow{name: "Memory", points: 20}
		// when
		got := r.isComment()
		// then
		assert.False(t, got)
	})
}

func TestCharacterAttributes(t *testing.T) {
	t.Run("shows attributes of exchanged character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAttributes(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAttributes,
		})
		factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{
			CharacterID:  c.ID,
			BonusRemaps:  optional.New[int64](2),
			Charisma:     17,
			Intelligence: 20,
			Memory:       21,
			Perception:   22,
			Willpower:    19,
		})
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		xassert.Equal(t, "Total points: 99", a.top.Text)
		xassert.Equal(t, widget.MediumImportance, a.top.Importance)
		got := xslices.Map(a.rows, func(r characterAttributeRow) characterAttributeRow {
			r.icon = nil
			return r
		})
		want := []characterAttributeRow{
			{name: "Perception", points: 22},
			{name: "Memory", points: 21},
			{name: "Willpower", points: 19},
			{name: "Intelligence", points: 20},
			{name: "Charisma", points: 17},
			{name: "Bonus Remaps Available: 2"},
		}
		xassert.Equal(t, want, got)
	})
	t.Run("shows no data when attributes are missing despite section data", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAttributes(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAttributes,
		})
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		xassert.Equal(t, "No data", a.top.Text)
		xassert.Equal(t, widget.WarningImportance, a.top.Importance)
		assert.Empty(t, a.rows)
	})
	t.Run("shows no character when character is exchanged to none", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAttributes(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.rows = []characterAttributeRow{{name: "Memory", points: 20}}
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), nil)
		// then
		xassert.Equal(t, "No character", a.top.Text)
		xassert.Equal(t, widget.LowImportance, a.top.Importance)
		assert.Empty(t, a.rows)
	})
	t.Run("shows no data when section was never updated", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAttributes(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		a.rows = []characterAttributeRow{{name: "Memory", points: 20}}
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		xassert.Equal(t, "No data", a.top.Text)
		xassert.Equal(t, widget.WarningImportance, a.top.Importance)
		assert.Empty(t, a.rows)
	})
	t.Run("shows error when section status can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAttributes(testdouble.NewUIFake(testdouble.UIParams{
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
		assert.Empty(t, a.rows)
	})
	t.Run("shows error when attributes can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAttributes(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAttributes,
		})
		if _, err := db.Exec("DROP TABLE character_attributes"); err != nil {
			t.Fatal(err)
		}
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		assert.Contains(t, a.top.Text, "ERROR:")
		xassert.Equal(t, widget.DangerImportance, a.top.Importance)
		assert.Empty(t, a.rows)
	})
	t.Run("updates when attributes of current character changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAttributes(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAttributes,
		})
		factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{CharacterID: c.ID})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAttributes,
		})
		// then
		assert.Contains(t, a.top.Text, "Total points:")
		assert.Len(t, a.rows, 6)
	})
	t.Run("ignores change of other character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAttributes(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAttributes,
		})
		factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{CharacterID: c.ID})
		a.character.Store(c)
		other := factory.CreateCharacterFull()
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: other.ID,
			Section:     app.SectionCharacterAttributes,
		})
		// then
		assert.Empty(t, a.top.Text)
		assert.Empty(t, a.rows)
	})
	t.Run("ignores change of other section", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterAttributes(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAttributes,
		})
		factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{CharacterID: c.ID})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		// then
		assert.Empty(t, a.top.Text)
		assert.Empty(t, a.rows)
	})
}
