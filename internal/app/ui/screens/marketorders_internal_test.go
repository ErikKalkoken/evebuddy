package screens

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	future := time.Now().Add(time.Hour)
	newRow := func(id int64, state app.MarketOrderState, typeName, locationName string) marketOrderRow {
		return marketOrderRow{
			expires:      future,
			locationName: locationName,
			orderID:      id,
			searchTarget: makeMarketOrderSearchTarget(typeName, locationName),
			state:        state,
			typeName:     typeName,
		}
	}
	rows := []marketOrderRow{
		newRow(1, app.OrderOpen, "Tritanium", "Jita IV - Moon 4"),
		newRow(2, app.OrderOpen, "Pyerite", "Amarr VIII (Oris)"),
		newRow(3, app.OrderCancelled, "Tritanium", "Amarr VIII (Oris)"),
	}
	ids := func(a *MarketOrders) []int64 {
		return xslices.Map(a.rowsFiltered, func(r marketOrderRow) int64 {
			return r.orderID
		})
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
		a.segmentState.SetSelected(slices.Index(marketOrderStateChoices, marketOrderStateHistory))
		a.filterChip.SetSelected(map[string]string{marketOrdersFilterType: "Tritanium"})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 3, a.rowsFiltered[0].orderID)
		}
	})
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("can search items mobile=%v", isMobile), func(t *testing.T) {
			a := newMarketOrders(t, isMobile)
			a.searchEntry.SetText("trit")
			assert.ElementsMatch(t, []int64{1}, ids(a))
		})
		t.Run(fmt.Sprintf("can search locations mobile=%v", isMobile), func(t *testing.T) {
			a := newMarketOrders(t, isMobile)
			a.searchEntry.SetText("amarr")
			assert.ElementsMatch(t, []int64{2}, ids(a))
		})
		t.Run(fmt.Sprintf("search ignores case mobile=%v", isMobile), func(t *testing.T) {
			a := newMarketOrders(t, isMobile)
			a.searchEntry.SetText("JITA")
			assert.ElementsMatch(t, []int64{1}, ids(a))
		})
		t.Run(fmt.Sprintf("can combine search and filter mobile=%v", isMobile), func(t *testing.T) {
			a := newMarketOrders(t, isMobile)
			if isMobile {
				a.filterChip.SetSelected(map[string]string{marketOrdersFilterType: "Pyerite"})
			} else {
				a.selectType.SetSelected("Pyerite")
			}
			a.searchEntry.SetText("jita")
			assert.Empty(t, ids(a))
		})
	}
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("can switch state mobile=%v", isMobile), func(t *testing.T) {
			a := newMarketOrders(t, isMobile)
			assert.Equal(t, marketOrderStateActive, a.currentFilter().state)
			if isMobile {
				a.segmentState.SetSelected(slices.Index(marketOrderStateChoices, marketOrderStateHistory))
			} else {
				a.selectState.SetSelected(marketOrderStateHistory)
			}
			assert.Equal(t, marketOrderStateHistory, a.currentFilter().state)
			assert.ElementsMatch(t, []int64{3}, ids(a))
		})
	}
	t.Run("search does not match across fields", func(t *testing.T) {
		a := newMarketOrders(t, false)
		a.searchEntry.SetText("tritaniumjita")
		assert.Empty(t, ids(a))
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
