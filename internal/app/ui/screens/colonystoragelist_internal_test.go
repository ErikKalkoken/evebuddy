package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestColonyStorageList(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	items := []colonyStorageItem{
		{typeID: 1, name: "Alpha", group: "Basic Commodities", quantity: 100, volume: 38},
		{typeID: 2, name: "Bravo", group: "Planet Solid - Raw Resource", quantity: 5_000, volume: 50},
		{typeID: 3, name: "Charlie", quantity: 2, volume: 200}, // unknown group
	}
	names := func(rows []colonyStorageItem) []string {
		var s []string
		for _, r := range rows {
			s = append(s, r.name)
		}
		return s
	}

	t.Run("should sort by volume descending by default", func(t *testing.T) {
		a := newColonyStorageList(u)
		a.set(items)
		assert.Equal(t, []string{"Charlie", "Bravo", "Alpha"}, names(a.rowsFiltered))
		assert.Equal(t, "3 items • 288.00 m3", a.summary.Text)
		assert.False(t, a.empty.Visible())
	})
	t.Run("should sort by other columns", func(t *testing.T) {
		a := newColonyStorageList(u)
		a.set(items)
		a.columnSorter.Set("Name", xwidget.SortAsc)
		a.filterRows()
		assert.Equal(t, []string{"Alpha", "Bravo", "Charlie"}, names(a.rowsFiltered))
		a.columnSorter.Set("Quantity", xwidget.SortDesc)
		a.filterRows()
		assert.Equal(t, []string{"Bravo", "Alpha", "Charlie"}, names(a.rowsFiltered))
	})
	t.Run("should filter by tier", func(t *testing.T) {
		a := newColonyStorageList(u)
		a.set(items)
		a.filterChip.SetSelected(map[string]string{colonyStorageFilterTier: "Basic Commodities"})
		a.filterRows()
		assert.Equal(t, []string{"Alpha"}, names(a.rowsFiltered))
		assert.Equal(t, "1 item • 38.00 m3", a.summary.Text)
	})
	t.Run("should filter types without group as other", func(t *testing.T) {
		a := newColonyStorageList(u)
		a.set(items)
		a.filterChip.SetSelected(map[string]string{colonyStorageFilterTier: colonyStorageTierOther})
		a.filterRows()
		assert.Equal(t, []string{"Charlie"}, names(a.rowsFiltered))
	})
	t.Run("should show empty when there are no items", func(t *testing.T) {
		a := newColonyStorageList(u)
		a.set([]colonyStorageItem{})
		assert.Empty(t, a.rowsFiltered)
		assert.True(t, a.empty.Visible())
		assert.Equal(t, "Empty", a.empty.Text)
		assert.Equal(t, "0 items • 0.00 m3", a.summary.Text)
	})
	t.Run("should keep showing items after update", func(t *testing.T) {
		a := newColonyStorageList(u)
		a.set([]colonyStorageItem{})
		a.set(items)
		assert.Len(t, a.rowsFiltered, 3)
		assert.False(t, a.empty.Visible())
	})
}

func TestColonyStorageItemDetails(t *testing.T) {
	x := colonyStorageItem{quantity: 4_553, volume: 45.53}
	assert.Equal(t, "4,553 units • 45.53 m3", colonyStorageItemDetails(x))
}
