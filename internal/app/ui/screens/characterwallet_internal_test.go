package screens

import (
	"context"
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
)

func TestCharacterWallet_UpdateBalance(t *testing.T) {
	t.Run("shows and reports balance", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.OnBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1234567.89),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		// when
		a.UpdateBalance(t.Context())
		// then
		xassert.Equal(t, "1,234,567.89 ISK", a.balanceLabel.Text)
		xassert.Equal(t, widget.MediumImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{optional.New(1234567.89)}, reported)
	})
	t.Run("shows nothing and reports empty balance without character", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.OnBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.balanceLabel.SetText("1,000 ISK")
		// when
		a.UpdateBalance(t.Context())
		// then
		assert.Empty(t, a.balanceLabel.Text)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("shows no data when section was never updated", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.OnBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.character.Store(factory.CreateCharacterFull())
		// when
		a.UpdateBalance(t.Context())
		// then
		xassert.Equal(t, "No data", a.balanceLabel.Text)
		xassert.Equal(t, widget.WarningImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("shows no data when character has no wallet balance", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.OnBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		// when
		a.UpdateBalance(t.Context())
		// then
		xassert.Equal(t, "No data", a.balanceLabel.Text)
		xassert.Equal(t, widget.WarningImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("shows error when section status can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.OnBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.character.Store(factory.CreateCharacterFull())
		if _, err := db.Exec("DROP TABLE character_section_status"); err != nil {
			t.Fatal(err)
		}
		// when
		a.UpdateBalance(t.Context())
		// then
		assert.Contains(t, a.balanceLabel.Text, "ERROR:")
		xassert.Equal(t, widget.DangerImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("keeps balance when update is canceled", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.OnBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.character.Store(factory.CreateCharacterFull())
		a.balanceLabel.SetText("1,000 ISK")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		// when
		a.UpdateBalance(ctx)
		// then
		xassert.Equal(t, "1,000 ISK", a.balanceLabel.Text)
		assert.Empty(t, reported)
	})
}

func TestCharacterWallet_Signals(t *testing.T) {
	t.Run("shows balance of exchanged character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1000.0),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		xassert.Equal(t, "1,000.00 ISK", a.balanceLabel.Text)
	})
	t.Run("updates balance when wallet balance of current character changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1000.0),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		// then
		xassert.Equal(t, "1,000.00 ISK", a.balanceLabel.Text)
	})
	t.Run("ignores change of other character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1000.0),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		other := factory.CreateCharacterFull()
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: other.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		// then
		assert.Empty(t, a.balanceLabel.Text)
	})
	t.Run("ignores change of other section", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1000.0),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		assert.Empty(t, a.balanceLabel.Text)
	})
}
