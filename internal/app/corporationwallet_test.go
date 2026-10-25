package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCorporationWalletJournalEntryRefTypeDisplay(t *testing.T) {
	we := app.CorporationWalletJournalEntry{RefType: "market_transaction"}
	xassert.Equal(t, "Market Transaction", we.RefTypeDisplay())
}

func TestCorporationWalletTransactionTotal(t *testing.T) {
	wt := &app.CorporationWalletTransaction{UnitPrice: 1.5, Quantity: 4}
	xassert.Equal(t, 6.0, wt.Total())
}
