package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestDivisionID(t *testing.T) {
	xassert.Equal(t, int64(1), app.Division1.ID())
}

func TestDivisionDefaultWalletName(t *testing.T) {
	xassert.Equal(t, "Master Wallet", app.Division1.DefaultWalletName())
	xassert.Equal(t, "", app.DivisionZero.DefaultWalletName())
}

func TestCorporationWalletJournalEntryRefTypeDisplay(t *testing.T) {
	we := app.CorporationWalletJournalEntry{RefType: "market_transaction"}
	xassert.Equal(t, "Market Transaction", we.RefTypeDisplay())
}

func TestCorporationWalletTransactionTotal(t *testing.T) {
	wt := &app.CorporationWalletTransaction{UnitPrice: 1.5, Quantity: 4}
	xassert.Equal(t, 6.0, wt.Total())
}

func TestCorporation_IDorZero(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var c *app.Corporation
		xassert.Equal(t, 0, c.IDOrZero())
	})
	t.Run("not nil", func(t *testing.T) {
		c := &app.Corporation{ID: 42}
		xassert.Equal(t, 42, c.IDOrZero())
	})
}

func TestCorporation_NameorZero(t *testing.T) {
	t.Run("corp is nil", func(t *testing.T) {
		var c *app.Corporation
		xassert.Equal(t, "", c.NameOrZero())
	})
	t.Run("corp is not nil", func(t *testing.T) {
		c := &app.Corporation{EveCorporation: &app.EveCorporation{Name: "Alpha"}}
		xassert.Equal(t, "Alpha", c.NameOrZero())
	})
	t.Run("eve corp is nil", func(t *testing.T) {
		c := &app.Corporation{}
		xassert.Equal(t, "", c.NameOrZero())
	})
}
