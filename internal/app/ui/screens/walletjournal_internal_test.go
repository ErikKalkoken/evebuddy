package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestWalletJournalFilter_Match(t *testing.T) {
	inflow := walletJournalRow{amount: optional.New(100.0), refTypeDisplay: "Bounty"}
	outflow := walletJournalRow{amount: optional.New(-50.0), refTypeDisplay: "Bounty"}
	unknown := walletJournalRow{refTypeDisplay: "Bounty"}
	for _, tc := range []struct {
		name   string
		filter walletJournalFilter
		row    walletJournalRow
		want   bool
	}{
		{"no filter", walletJournalFilter{}, inflow, true},
		{"type matches", walletJournalFilter{refType: "Bounty"}, inflow, true},
		{"type differs", walletJournalFilter{refType: "Market Transaction"}, inflow, false},
		{"inflow matches", walletJournalFilter{direction: walletJournalDirectionInflow}, inflow, true},
		{"inflow but outflow", walletJournalFilter{direction: walletJournalDirectionInflow}, outflow, false},
		{"outflow matches", walletJournalFilter{direction: walletJournalDirectionOutflow}, outflow, true},
		{"outflow but inflow", walletJournalFilter{direction: walletJournalDirectionOutflow}, inflow, false},
		{"inflow but amount unknown", walletJournalFilter{direction: walletJournalDirectionInflow}, unknown, false},
		{"outflow but amount unknown", walletJournalFilter{direction: walletJournalDirectionOutflow}, unknown, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestWalletJournalRow_SetSearchTarget(t *testing.T) {
	r := walletJournalRow{description: "Alpha deposited cash", reason: optional.New("For Fuel")}
	r.setSearchTarget()
	assert.Equal(t, "alpha deposited cash\nfor fuel", r.searchTarget)
}

func TestWalletJournal_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []walletJournalRow{
		{refID: 1, amount: optional.New(100.0), refTypeDisplay: "Bounty", searchTarget: "bounty prizes"},
		{refID: 2, amount: optional.New(-50.0), refTypeDisplay: "Market Transaction", searchTarget: "market escrow"},
	}
	newWalletJournal := func(t *testing.T, isMobile bool) *WalletJournal {
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	refIDs := func(a *WalletJournal) []int64 {
		return xslices.Map(a.rowsFiltered, func(r walletJournalRow) int64 {
			return r.refID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newWalletJournal(t, true)
		a.filterChip.SetSelected(map[string]string{walletJournalFilterType: "Bounty"})
		assert.ElementsMatch(t, []int64{1}, refIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newWalletJournal(t, false)
		a.selectType.SetSelected("Bounty")
		assert.ElementsMatch(t, []int64{1}, refIDs(a))
	})
	t.Run("can filter direction on mobile", func(t *testing.T) {
		a := newWalletJournal(t, true)
		a.filterChip.SetSelected(map[string]string{walletJournalFilterDirection: walletJournalDirectionOutflow})
		assert.ElementsMatch(t, []int64{2}, refIDs(a))
	})
	t.Run("can filter direction on desktop", func(t *testing.T) {
		a := newWalletJournal(t, false)
		a.selectDirection.SetSelected(walletJournalDirectionOutflow)
		assert.ElementsMatch(t, []int64{2}, refIDs(a))
	})
	t.Run("can search", func(t *testing.T) {
		a := newWalletJournal(t, true)
		a.searchEntry.SetText("escrow")
		assert.ElementsMatch(t, []int64{2}, refIDs(a))
	})
}
