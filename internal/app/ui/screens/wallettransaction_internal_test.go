package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestWalletTransactionFilter_Match(t *testing.T) {
	r := walletTransactionRow{
		categoryName: "Ship",
		clientName:   "Bruce",
		isBuy:        true,
		locationName: "Jita IV - Moon 4",
		regionName:   "The Forge",
		typeName:     "Merlin",
	}
	sell := r
	sell.isBuy = false
	for _, tc := range []struct {
		name   string
		filter walletTransactionFilter
		row    walletTransactionRow
		want   bool
	}{
		{"no filter", walletTransactionFilter{}, r, true},
		{"buy matches", walletTransactionFilter{activity: marketTransactionActivityBuy}, r, true},
		{"buy but is sell", walletTransactionFilter{activity: marketTransactionActivityBuy}, sell, false},
		{"sell matches", walletTransactionFilter{activity: marketTransactionActivitySell}, sell, true},
		{"sell but is buy", walletTransactionFilter{activity: marketTransactionActivitySell}, r, false},
		{"category matches", walletTransactionFilter{category: "Ship"}, r, true},
		{"category differs", walletTransactionFilter{category: "Module"}, r, false},
		{"client differs", walletTransactionFilter{client: "Alice"}, r, false},
		{"location differs", walletTransactionFilter{location: "Amarr"}, r, false},
		{"region differs", walletTransactionFilter{region: "Domain"}, r, false},
		{"type differs", walletTransactionFilter{typeName: "Rifter"}, r, false},
		{"all match", walletTransactionFilter{activity: marketTransactionActivityBuy, category: "Ship", client: "Bruce", location: "Jita IV - Moon 4", region: "The Forge", typeName: "Merlin"}, r, true},
		{"one of many differs", walletTransactionFilter{category: "Ship", typeName: "Rifter"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestWalletTransactions_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []walletTransactionRow{
		{transactionID: 1, categoryName: "Ship", clientName: "Bruce", isBuy: true, typeName: "Merlin"},
		{transactionID: 2, categoryName: "Module", clientName: "Alice", isBuy: false, typeName: "Damage Control"},
	}
	for i := range rows {
		rows[i].setSearchTarget()
	}
	newWalletTransactions := func(t *testing.T, isMobile bool) *WalletTransactions {
		a := NewCharacterWalletTransaction(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newWalletTransactions(t, true)
		a.filterChip.SetSelected(map[string]string{walletTransactionFilterActivity: marketTransactionActivitySell})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 2, a.rowsFiltered[0].transactionID)
		}
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newWalletTransactions(t, false)
		a.selectActivity.SetSelected(marketTransactionActivitySell)
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 2, a.rowsFiltered[0].transactionID)
		}
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newWalletTransactions(t, true)
		assert.Equal(t, map[string]string{
			walletTransactionFilterActivity: "",
			walletTransactionFilterCategory: "",
			walletTransactionFilterClient:   "",
			walletTransactionFilterLocation: "",
			walletTransactionFilterRegion:   "",
			walletTransactionFilterType:     "",
		}, a.filterChip.Selected())
	})
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("can search client and type names mobile=%v", isMobile), func(t *testing.T) {
			for _, tc := range []struct {
				search string
				want   []int64
			}{
				{"bru", []int64{1}},
				{"DAMAGE", []int64{2}},
				{"merlin", []int64{1}},
				{"e", []int64{1, 2}}, // too short to search
				{"xyz", []int64{}},
			} {
				a := newWalletTransactions(t, isMobile)
				a.searchEntry.SetText(tc.search)
				got := xslices.Map(a.rowsFiltered, func(r walletTransactionRow) int64 {
					return r.transactionID
				})
				assert.ElementsMatch(t, tc.want, got, tc.search)
			}
		})
	}
}
