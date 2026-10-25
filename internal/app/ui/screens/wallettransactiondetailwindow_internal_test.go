package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestShowCharacterWalletTransactionWindowAsync(t *testing.T) {
	t.Run("can show buy transaction", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		o := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{IsBuy: true})
		// when
		ShowCharacterWalletTransactionWindowAsync(u, o.CharacterID, "Bruce", o.TransactionID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", o.CharacterID, o.TransactionID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show sell transaction", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		o := factory.CreateCharacterWalletTransaction()
		// when
		ShowCharacterWalletTransactionWindowAsync(u, o.CharacterID, "Bruce", o.TransactionID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", o.CharacterID, o.TransactionID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("closes window when transaction can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		// when
		ShowCharacterWalletTransactionWindowAsync(u, c.ID, "Bruce", 42)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", c.ID, 42))
		assert.True(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", 7, 42))
		// when
		ShowCharacterWalletTransactionWindowAsync(u, 7, "Bruce", 42)
		// then
		assert.Nil(t, w.Content())
	})
}

func TestShowCorporationWalletTransactionWindowAsync(t *testing.T) {
	t.Run("can show buy transaction", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletTransactions1)
		o := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      1,
			IsBuy:         true,
		})
		// when
		ShowCorporationWalletTransactionWindowAsync(u, c.ID, "Wayne Enterprises", app.Division1, o.TransactionID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", c.ID, o.TransactionID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show sell transaction", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletTransactions1)
		o := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      1,
		})
		// when
		ShowCorporationWalletTransactionWindowAsync(u, c.ID, "Wayne Enterprises", app.Division1, o.TransactionID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", c.ID, o.TransactionID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("closes window when transaction can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletTransactions1)
		// when
		ShowCorporationWalletTransactionWindowAsync(u, c.ID, "Wayne Enterprises", app.Division1, 42)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", c.ID, 42))
		assert.True(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("wallettransaction-%d-%d", 7, 42))
		// when
		ShowCorporationWalletTransactionWindowAsync(u, 7, "Wayne Enterprises", app.Division1, 42)
		// then
		assert.Nil(t, w.Content())
	})
}
