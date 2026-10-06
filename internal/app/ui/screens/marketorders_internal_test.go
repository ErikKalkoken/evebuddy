package screens

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

func TestMarketOrdersFilter_Match(t *testing.T) {
	future := time.Now().Add(time.Hour)
	r := marketOrderRow{
		characterName: "Bruce",
		expires:       future,
		ownerName:     "Bruce",
		regionName:    "The Forge",
		state:         app.OrderOpen,
		tags:          set.Of("alpha"),
		typeName:      "Tritanium",
	}
	expired := r
	expired.expires = time.Now().Add(-time.Hour) // open but past expiry
	cancelled := r
	cancelled.state = app.OrderCancelled
	corpOrder := r
	corpOrder.ownerName = "Wayne Inc" // corporation orders are owned by the corporation
	for _, tc := range []struct {
		name   string
		filter marketOrdersFilter
		row    marketOrderRow
		want   bool
	}{
		{"no filter", marketOrdersFilter{}, r, true},
		{"owner matches", marketOrdersFilter{owner: "Bruce"}, r, true},
		{"owner differs", marketOrdersFilter{owner: "Alice"}, r, false},
		{"corporation owner matches", marketOrdersFilter{owner: "Wayne Inc"}, corpOrder, true},
		{"character does not match own corporation order", marketOrdersFilter{owner: "Bruce"}, corpOrder, false},
		{"region differs", marketOrdersFilter{region: "Domain"}, r, false},
		{"tag matches", marketOrdersFilter{tag: "alpha"}, r, true},
		{"tag missing", marketOrdersFilter{tag: "bravo"}, r, false},
		{"type differs", marketOrdersFilter{typeName: "Pyerite"}, r, false},
		{"active matches", marketOrdersFilter{state: marketOrderStateActive}, r, true},
		{"active but expired", marketOrdersFilter{state: marketOrderStateActive}, expired, false},
		{"active but cancelled", marketOrdersFilter{state: marketOrderStateActive}, cancelled, false},
		{"history matches expired", marketOrdersFilter{state: marketOrderStateHistory}, expired, true},
		{"history matches cancelled", marketOrdersFilter{state: marketOrderStateHistory}, cancelled, true},
		{"history but open", marketOrdersFilter{state: marketOrderStateHistory}, r, false},
		{"all match", marketOrdersFilter{owner: "Bruce", region: "The Forge", state: marketOrderStateActive, tag: "alpha", typeName: "Tritanium"}, r, true},
		{"one of many differs", marketOrdersFilter{owner: "Bruce", typeName: "Pyerite"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestMarketOrders_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	future := time.Now().Add(time.Hour)
	rows := []marketOrderRow{
		{orderID: 1, expires: future, state: app.OrderOpen, typeName: "Tritanium"},
		{orderID: 2, expires: future, state: app.OrderOpen, typeName: "Pyerite"},
		{orderID: 3, expires: future, state: app.OrderCancelled, typeName: "Tritanium"},
	}
	newMarketOrders := func(t *testing.T, isMobile bool) *MarketOrders {
		a := NewMarketOrders(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}), true)
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2) // active by default
		return a
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newMarketOrders(t, true)
		a.filterChip.SetSelected(map[string]string{marketOrdersFilterType: "Tritanium"})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 1, a.rowsFiltered[0].orderID)
		}
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newMarketOrders(t, false)
		a.selectType.SetSelected("Tritanium")
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 1, a.rowsFiltered[0].orderID)
		}
	})
	t.Run("can combine state and filter on mobile", func(t *testing.T) {
		a := newMarketOrders(t, true)
		a.selectState.SetSelected(marketOrderStateHistory)
		a.filterChip.SetSelected(map[string]string{marketOrdersFilterType: "Tritanium"})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 3, a.rowsFiltered[0].orderID)
		}
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newMarketOrders(t, true)
		assert.Equal(t, map[string]string{
			marketOrdersFilterOwner:  "",
			marketOrdersFilterRegion: "",
			marketOrdersFilterTag:    "",
			marketOrdersFilterType:   "",
		}, a.filterChip.Selected())
	})
}
