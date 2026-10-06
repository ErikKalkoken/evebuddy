package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestFlyableShipsFilter_Match(t *testing.T) {
	r := flyableShipRow{canFly: true, groupName: "Frigate"}
	cannot := r
	cannot.canFly = false
	for _, tc := range []struct {
		name   string
		filter flyableShipsFilter
		row    flyableShipRow
		want   bool
	}{
		{"no filter", flyableShipsFilter{}, r, true},
		{"class matches", flyableShipsFilter{class: "Frigate"}, r, true},
		{"class differs", flyableShipsFilter{class: "Cruiser"}, r, false},
		{"can fly matches", flyableShipsFilter{flyable: flyableCan}, r, true},
		{"can fly but cannot", flyableShipsFilter{flyable: flyableCan}, cannot, false},
		{"cannot fly matches", flyableShipsFilter{flyable: flyableCanNot}, cannot, true},
		{"cannot fly but can", flyableShipsFilter{flyable: flyableCanNot}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestFlyableShips_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []flyableShipRow{
		{typeID: 1, typeName: "Merlin", groupName: "Frigate", canFly: true, searchText: "merlin frigate"},
		{typeID: 2, typeName: "Caracal", groupName: "Cruiser", canFly: false, searchText: "caracal cruiser"},
	}
	newFlyableShips := func(t *testing.T, isMobile bool) *FlyableShips {
		a := NewFlyableShips(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync()
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	typeIDs := func(a *FlyableShips) []int64 {
		return xslices.Map(a.rowsFiltered, func(r flyableShipRow) int64 {
			return r.typeID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newFlyableShips(t, true)
		a.filterChip.SetSelected(map[string]string{flyableShipsFilterFlyable: flyableCan})
		assert.ElementsMatch(t, []int64{1}, typeIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newFlyableShips(t, false)
		a.selectFlyable.SetSelected(flyableCan)
		assert.ElementsMatch(t, []int64{1}, typeIDs(a))
	})
	t.Run("can search on mobile", func(t *testing.T) {
		a := newFlyableShips(t, true)
		a.searchEntry.SetText("cruis")
		assert.ElementsMatch(t, []int64{2}, typeIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newFlyableShips(t, true)
		assert.Equal(t, map[string]string{
			flyableShipsFilterClass:   "",
			flyableShipsFilterFlyable: "",
		}, a.filterChip.Selected())
	})
	t.Run("resets filters when character changes on mobile", func(t *testing.T) {
		a := newFlyableShips(t, true)
		a.filterChip.SetSelected(map[string]string{flyableShipsFilterFlyable: flyableCan})
		require.True(t, a.filterChip.IsOn())

		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), factory.CreateCharacter())

		assert.False(t, a.filterChip.IsOn())
	})
}
