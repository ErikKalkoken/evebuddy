package screens

import (
	"database/sql"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestIndustrySlotsFilter_Match(t *testing.T) {
	r := industrySlotRow{free: 2, tags: set.Of("alpha")}
	full := r
	full.free = 0
	for _, tc := range []struct {
		name   string
		filter industrySlotsFilter
		row    industrySlotRow
		want   bool
	}{
		{"no filter", industrySlotsFilter{}, r, true},
		{"has free slots matches", industrySlotsFilter{freeSlots: industrySlotsFreeSome}, r, true},
		{"has free slots but full", industrySlotsFilter{freeSlots: industrySlotsFreeSome}, full, false},
		{"no free slots matches", industrySlotsFilter{freeSlots: industrySlotsFreeNone}, full, true},
		{"no free slots but has", industrySlotsFilter{freeSlots: industrySlotsFreeNone}, r, false},
		{"tag matches", industrySlotsFilter{tag: "alpha"}, r, true},
		{"tag missing", industrySlotsFilter{tag: "bravo"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestIndustrySlots_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []industrySlotRow{
		{characterID: 1, free: 2, total: 2},
		{characterID: 2, free: 0, busy: 2, total: 2},
	}
	newIndustrySlots := func(t *testing.T, isMobile bool) *IndustrySlots {
		a := NewIndustrySlots(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}), app.ManufacturingJob)
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 3) // incl. totals row
		return a
	}
	characterIDs := func(a *IndustrySlots) []int64 {
		var rows []industrySlotRow
		for _, r := range a.rowsFiltered {
			if !r.isTotal {
				rows = append(rows, r)
			}
		}
		return xslices.Map(rows, func(r industrySlotRow) int64 {
			return r.characterID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newIndustrySlots(t, true)
		a.filterChip.SetSelected(map[string]string{industrySlotsFilterFreeSlots: industrySlotsFreeNone})
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newIndustrySlots(t, false)
		a.selectFreeSlots.SetSelected(industrySlotsFreeNone)
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newIndustrySlots(t, true)
		assert.Equal(t, map[string]string{
			industrySlotsFilterFreeSlots: "",
			industrySlotsFilterTag:       "",
		}, a.filterChip.Selected())
	})
}

func TestIndustrySlots_ErrorDisplay(t *testing.T) {
	newScreen := func(t *testing.T) (*IndustrySlots, *sql.DB) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		a := NewIndustrySlots(u, app.ManufacturingJob)
		return a, db
	}
	t.Run("shows error and clears rows when update fails and keeps error on filtering", func(t *testing.T) {
		a, db := newScreen(t)
		a.rows = []industrySlotRow{{free: 2}}
		a.filterRowsAsync("")
		if _, err := db.Exec("DROP TABLE character_industry_jobs"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")
		assert.Empty(t, a.rows)

		a.filterRowsAsync("")
		assert.True(t, a.errorLabel.Visible())
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		a, _ := newScreen(t)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.update(t.Context())
		assert.False(t, a.errorLabel.Visible())
	})
}
