package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCorporationOverviewFilter_Match(t *testing.T) {
	r := corporationOverviewRow{
		alliance: optional.New(&app.EveEntity{Name: "Goonswarm"}),
		faction:  optional.New(&app.EveEntity{Name: "Caldari State"}),
	}
	for _, tc := range []struct {
		name   string
		filter corporationOverviewFilter
		want   bool
	}{
		{"no filter", corporationOverviewFilter{}, true},
		{"alliance matches", corporationOverviewFilter{alliance: "Goonswarm"}, true},
		{"alliance differs", corporationOverviewFilter{alliance: "Test"}, false},
		{"faction matches", corporationOverviewFilter{faction: "Caldari State"}, true},
		{"faction differs", corporationOverviewFilter{faction: "Amarr Empire"}, false},
		{"both match", corporationOverviewFilter{alliance: "Goonswarm", faction: "Caldari State"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(r))
		})
	}
}

func TestCorporationOverview_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []corporationOverviewRow{
		{corporationID: 1, name: "Wayne Inc", alliance: optional.New(&app.EveEntity{Name: "Goonswarm"}), searchTarget: "wayne inc"},
		{corporationID: 2, name: "Acme", searchTarget: "acme"},
	}
	newOverview := func(t *testing.T, isMobile bool) *CorporationOverview {
		a := NewCorporationOverview(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	corporationIDs := func(a *CorporationOverview) []int64 {
		return xslices.Map(a.rowsFiltered, func(r corporationOverviewRow) int64 {
			return r.corporationID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newOverview(t, true)
		a.filterChip.SetSelected(map[string]string{corporationOverviewFilterAlliance: "Goonswarm"})
		assert.ElementsMatch(t, []int64{1}, corporationIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newOverview(t, false)
		a.selectAlliance.SetSelected("Goonswarm")
		assert.ElementsMatch(t, []int64{1}, corporationIDs(a))
	})
	t.Run("can search on mobile", func(t *testing.T) {
		a := newOverview(t, true)
		a.searchEntry.SetText("acm")
		assert.ElementsMatch(t, []int64{2}, corporationIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newOverview(t, true)
		assert.Equal(t, map[string]string{
			corporationOverviewFilterAlliance: "",
			corporationOverviewFilterFaction:  "",
		}, a.filterChip.Selected())
	})
}
