package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestColonyRouteList(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	items := []colonyRouteItem{
		{name: "Water", quantity: 20, otherName: "A"},
		{name: "Base Metals", quantity: 3_000, otherName: "B"},
		{name: "Water", quantity: 5, otherName: "C", isIncoming: true},
		{name: "Base Metals", quantity: 10_000, otherName: "C"},
		{name: "Base Metals", quantity: 3_000, otherName: "A"},
	}
	details := func(a *colonyRouteList) []string {
		var s []string
		for _, x := range a.rowsFiltered {
			s = append(s, x.name+": "+colonyRouteItemDetails(x))
		}
		return s
	}

	t.Run("should sort by item by default, then incoming first, larger quantity and installation", func(t *testing.T) {
		a := newColonyRouteList(u)
		a.set(items)
		assert.Equal(t, []string{
			"Base Metals: 10,000 units to C",
			"Base Metals: 3,000 units to A",
			"Base Metals: 3,000 units to B",
			"Water: 5 units from C",
			"Water: 20 units to A",
		}, details(a))
		assert.Equal(t, "5 routes", a.summary.Text)
		assert.False(t, a.empty.Visible())
	})
	t.Run("should sort by other columns", func(t *testing.T) {
		a := newColonyRouteList(u)
		a.set(items)
		a.columnSorter.Set("Quantity", xwidget.SortAsc)
		a.filterRows()
		assert.Equal(t, []string{
			"Water: 5 units from C",
			"Water: 20 units to A",
			"Base Metals: 3,000 units to A",
			"Base Metals: 3,000 units to B",
			"Base Metals: 10,000 units to C",
		}, details(a))
		a.columnSorter.Set("Installation", xwidget.SortAsc)
		a.filterRows()
		assert.Equal(t, []string{
			"Base Metals: 3,000 units to A",
			"Water: 20 units to A",
			"Base Metals: 3,000 units to B",
			"Base Metals: 10,000 units to C",
			"Water: 5 units from C",
		}, details(a))
	})
	t.Run("should filter by direction", func(t *testing.T) {
		a := newColonyRouteList(u)
		a.set(items)
		a.filterChip.SetSelected(map[string]string{colonyRouteFilterDirection: colonyRouteIncoming})
		a.filterRows()
		assert.Equal(t, []string{"Water: 5 units from C"}, details(a))
		assert.Equal(t, "1 route", a.summary.Text)
	})
	t.Run("should filter by item", func(t *testing.T) {
		a := newColonyRouteList(u)
		a.set(items)
		a.filterChip.SetSelected(map[string]string{colonyRouteFilterItem: "Water"})
		a.filterRows()
		assert.Equal(t, []string{"Water: 5 units from C", "Water: 20 units to A"}, details(a))
	})
	t.Run("should show when no routes match the filter", func(t *testing.T) {
		a := newColonyRouteList(u)
		a.set(items)
		a.filterChip.SetSelected(map[string]string{colonyRouteFilterDirection: colonyRouteIncoming, colonyRouteFilterItem: "Base Metals"})
		a.filterRows()
		assert.Empty(t, a.rowsFiltered)
		assert.True(t, a.empty.Visible())
		assert.Equal(t, "No matching routes", a.empty.Text)
	})
	t.Run("should show unknown installation", func(t *testing.T) {
		x := colonyRouteItem{name: "Water", quantity: 1_000}
		assert.Equal(t, "1,000 units to unknown installation", colonyRouteItemDetails(x))
	})
	t.Run("should show connected installation when selected", func(t *testing.T) {
		var called bool
		a := newColonyRouteList(u)
		a.set([]colonyRouteItem{{name: "Water", onSelected: func() { called = true }}})
		a.list.Select(0)
		assert.True(t, called)
	})
	t.Run("should ignore selecting route to unknown installation", func(t *testing.T) {
		a := newColonyRouteList(u)
		a.set([]colonyRouteItem{{name: "Water"}})
		assert.NotPanics(t, func() {
			a.list.Select(0)
		})
	})
	t.Run("should show empty text when there are no routes", func(t *testing.T) {
		a := newColonyRouteList(u)
		a.set([]colonyRouteItem{})
		assert.True(t, a.empty.Visible())
		assert.Equal(t, "No routes", a.empty.Text)
		assert.Equal(t, "0 routes", a.summary.Text)
	})
}
