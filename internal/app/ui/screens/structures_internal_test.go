package screens

import (
	"database/sql"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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

func TestStructureRow_StateText(t *testing.T) {
	t.Run("no timer", func(t *testing.T) {
		r := structureRow{stateShort: "Online"}
		assert.Equal(t, "Online", r.stateText())
	})
	t.Run("counts down and expires", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := structureRow{stateShort: "Armor", stateTimerEnd: optional.New(time.Now().Add(3 * time.Hour))}
			assert.Equal(t, "Armor: 3h 0m", r.stateText())

			time.Sleep(time.Hour)
			assert.Equal(t, "Armor: 2h 0m", r.stateText())

			time.Sleep(2 * time.Hour)
			assert.Equal(t, "Armor: EXPIRED", r.stateText()) // exactly at the end
		})
	})
}

func TestStructureRow_FuelExpiresDisplay(t *testing.T) {
	display := func(r structureRow) (string, fyne.ThemeColorName) {
		segs := r.fuelExpiresDisplay()
		require.Len(t, segs, 1)
		s := segs[0].(*widget.TextSegment)
		return s.Text, s.Style.ColorName
	}
	t.Run("low power", func(t *testing.T) {
		text, color := display(structureRow{})
		assert.Equal(t, "Low Power", text)
		assert.Equal(t, theme.ColorNameWarning, color)
	})
	t.Run("counts down and expires", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := structureRow{fuelExpires: optional.New(time.Now().Add(3 * time.Hour))}
			text, color := display(r)
			assert.Equal(t, "3h 0m", text)
			assert.Equal(t, theme.ColorNameForeground, color)

			time.Sleep(3 * time.Hour)
			text, color = display(r)
			assert.Equal(t, "EXPIRED", text)
			assert.Equal(t, theme.ColorNameWarning, color)
		})
	})
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
	c := factory.CreateCorporation()
	factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{CorporationID: c.ID})
	for _, tc := range []struct {
		name           string
		new            func(u baseUI) *Structures
		forCorporation bool
	}{
		{"corporation", NewStructuresForCorporation, true},
		{"unified", NewUnifiedStructures, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := testdouble.NewUIFake(testdouble.UIParams{
				App:     test.NewTempApp(t),
				Storage: st,
			})
			a := tc.new(u)
			if tc.forCorporation {
				a.corporation.Store(c)
			}
			require.Equal(t, 1, u.Signals().RefreshTickerExpired.Len())

			u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})

			assert.Empty(t, a.rows) // ticker only redraws, it must not reload from storage
		})
	}
}

func TestStructures_ErrorDisplay(t *testing.T) {
	newScreen := func(t *testing.T) (*Structures, *sql.DB) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		a := NewUnifiedStructures(u)
		return a, db
	}
	t.Run("shows error when update fails and keeps it on filtering", func(t *testing.T) {
		a, db := newScreen(t)
		if _, err := db.Exec("DROP TABLE corporation_structures"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")

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
