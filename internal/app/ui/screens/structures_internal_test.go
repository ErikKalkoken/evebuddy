package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestStructuresFilter_Match(t *testing.T) {
	r := structureRow{
		corporationName: "Wayne Inc",
		isFullPower:     true,
		regionName:      "The Forge",
		services:        set.Of("Market"),
		solarSystemName: "Jita",
		stateDisplay:    "Shield vulnerable",
		typeName:        "Keepstar",
	}
	lowPower := r
	lowPower.isFullPower = false
	for _, tc := range []struct {
		name   string
		filter structuresFilter
		row    structureRow
		want   bool
	}{
		{"no filter", structuresFilter{}, r, true},
		{"owner matches", structuresFilter{owner: "Wayne Inc"}, r, true},
		{"owner differs", structuresFilter{owner: "Other"}, r, false},
		{"high power matches", structuresFilter{power: structuresPowerHigh}, r, true},
		{"high power but low", structuresFilter{power: structuresPowerHigh}, lowPower, false},
		{"low power matches", structuresFilter{power: structuresPowerLow}, lowPower, true},
		{"low power but high", structuresFilter{power: structuresPowerLow}, r, false},
		{"region differs", structuresFilter{region: "Domain"}, r, false},
		{"service matches", structuresFilter{service: "Market"}, r, true},
		{"service missing", structuresFilter{service: "Clone Bay"}, r, false},
		{"system differs", structuresFilter{solarSystem: "Amarr"}, r, false},
		{"state differs", structuresFilter{state: "Armor reinforce"}, r, false},
		{"type differs", structuresFilter{typeName: "Fortizar"}, r, false},
		{"all match", structuresFilter{owner: "Wayne Inc", power: structuresPowerHigh, region: "The Forge", service: "Market", solarSystem: "Jita", state: "Shield vulnerable", typeName: "Keepstar"}, r, true},
		{"one of many differs", structuresFilter{owner: "Wayne Inc", typeName: "Fortizar"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestStructures_Filter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []structureRow{
		{structureID: 1, structureName: "Jita - Trade Hub", solarSystemName: "Jita", typeName: "Keepstar"},
		{structureID: 2, structureName: "Home Sweet Home", solarSystemName: "Amarr", typeName: "Astrahus"},
	}
	for i := range rows {
		rows[i].setSearchTarget()
	}
	newStructures := func(t *testing.T, isMobile, forCorporation bool) *Structures {
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		})
		var a *Structures
		if forCorporation {
			a = NewStructuresForCorporation(u)
		} else {
			a = NewUnifiedStructures(u)
		}
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newStructures(t, true, false)
		a.filterChip.SetSelected(map[string]string{structuresFilterType: "Astrahus"})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 2, a.rowsFiltered[0].structureID)
		}
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newStructures(t, false, false)
		a.selectType.SetSelected("Astrahus")
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 2, a.rowsFiltered[0].structureID)
		}
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newStructures(t, true, false)
		assert.Equal(t, map[string]string{
			structuresFilterOwner:   "",
			structuresFilterPower:   "",
			structuresFilterRegion:  "",
			structuresFilterService: "",
			structuresFilterState:   "",
			structuresFilterSystem:  "",
			structuresFilterType:    "",
		}, a.filterChip.Selected())
	})
	t.Run("hides owner filter for corporation on mobile", func(t *testing.T) {
		a := newStructures(t, true, true)
		assert.Equal(t, map[string]string{
			structuresFilterPower:   "",
			structuresFilterRegion:  "",
			structuresFilterService: "",
			structuresFilterState:   "",
			structuresFilterSystem:  "",
			structuresFilterType:    "",
		}, a.filterChip.Selected())
	})
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("can search structure and system names mobile=%v", isMobile), func(t *testing.T) {
			for _, tc := range []struct {
				search string
				want   []int64
			}{
				{"sweet", []int64{2}},
				{"AMARR", []int64{2}},
				{"jita", []int64{1}},
				{"e", []int64{1, 2}}, // too short to search
				{"xyz", []int64{}},
			} {
				a := newStructures(t, isMobile, false)
				a.searchEntry.SetText(tc.search)
				got := xslices.Map(a.rowsFiltered, func(r structureRow) int64 {
					return r.structureID
				})
				assert.ElementsMatch(t, tc.want, got, tc.search)
			}
		})
	}
	t.Run("resets filters and search when corporation changes on mobile", func(t *testing.T) {
		a := newStructures(t, true, true)
		a.filterChip.SetSelected(map[string]string{structuresFilterType: "Astrahus"})
		a.searchEntry.SetText("home")
		require.True(t, a.filterChip.IsOn())

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), factory.CreateCorporation())

		assert.False(t, a.filterChip.IsOn())
		assert.Empty(t, a.searchEntry.Text)
	})
	t.Run("resets filters and search when corporation changes on desktop", func(t *testing.T) {
		a := newStructures(t, false, true)
		a.selectType.SetSelected("Astrahus")
		a.searchEntry.SetText("home")

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), factory.CreateCorporation())

		assert.Empty(t, a.currentFilter())
		assert.Empty(t, a.searchEntry.Text)
	})
}

func TestStructures_RefreshTicker(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	c := factory.CreateCorporation()
	factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{CorporationID: c.ID})
	a := NewStructuresForCorporation(u)
	a.corporation.Store(c)

	u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})

	assert.Len(t, a.rows, 1)
}
