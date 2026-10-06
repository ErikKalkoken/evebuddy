package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestContractSlotsFilter_Match(t *testing.T) {
	r := contractSlotRow{corporationName: "Wayne Inc", free: 2, tags: set.Of("alpha")}
	full := r
	full.free = 0
	for _, tc := range []struct {
		name   string
		filter contractSlotsFilter
		row    contractSlotRow
		want   bool
	}{
		{"no filter", contractSlotsFilter{}, r, true},
		{"corporation matches", contractSlotsFilter{corporation: "Wayne Inc"}, r, true},
		{"corporation differs", contractSlotsFilter{corporation: "Other"}, r, false},
		{"has free slots matches", contractSlotsFilter{freeSlots: contractSlotsFreeSome}, r, true},
		{"has free slots but full", contractSlotsFilter{freeSlots: contractSlotsFreeSome}, full, false},
		{"no free slots matches", contractSlotsFilter{freeSlots: contractSlotsFreeNone}, full, true},
		{"no free slots but has", contractSlotsFilter{freeSlots: contractSlotsFreeNone}, r, false},
		{"tag matches", contractSlotsFilter{tag: "alpha"}, r, true},
		{"tag missing", contractSlotsFilter{tag: "bravo"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestContractSlots_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []contractSlotRow{
		{characterID: 1, free: 2, total: 2},
		{characterID: 2, free: 0, used: 2, total: 2},
	}
	newContractSlots := func(t *testing.T, isMobile bool) *ContractSlots {
		a := NewContractSlots(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}), false)
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 3) // incl. totals row
		return a
	}
	characterIDs := func(a *ContractSlots) []int64 {
		var rows []contractSlotRow
		for _, r := range a.rowsFiltered {
			if !r.isTotal {
				rows = append(rows, r)
			}
		}
		return xslices.Map(rows, func(r contractSlotRow) int64 {
			return r.characterID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newContractSlots(t, true)
		a.filterChip.SetSelected(map[string]string{contractSlotsFilterFreeSlots: contractSlotsFreeNone})
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newContractSlots(t, false)
		a.selectFreeSlots.SetSelected(contractSlotsFreeNone)
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newContractSlots(t, true)
		assert.Equal(t, map[string]string{
			contractSlotsFilterCorporation: "",
			contractSlotsFilterFreeSlots:   "",
			contractSlotsFilterTag:         "",
		}, a.filterChip.Selected())
	})
}
