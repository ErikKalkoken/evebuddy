package screens

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestColonies(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	now := time.Now().UTC()
	lastUpdate := now.Add(-65 * time.Minute)
	cp := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{LastUpdate: lastUpdate})
	ecuGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupExtractorControlUnits})
	ecuType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: ecuGroup.ID})
	storageGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupStorageFacilities})
	storageType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: storageGroup.ID})
	product := factory.CreateEveType(storage.CreateEveTypeParams{Volume: optional.New(0.01)})
	expiry := lastUpdate.Add(4 * time.Hour)
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID:      cp.ID,
		PinID:                  1,
		TypeID:                 ecuType.ID,
		ExtractorProductTypeID: optional.New(product.ID),
		ExtractorQtyPerCycle:   optional.New[int64](1081),
		ExtractorCycleTime:     optional.New(30 * time.Minute),
		InstallTime:            optional.New(lastUpdate),
		ExpiryTime:             optional.New(expiry),
		LastCycleStart:         optional.New(lastUpdate),
	})
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID: cp.ID,
		PinID:             2,
		TypeID:            storageType.ID,
	})
	factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{
		CharacterPlanetID: cp.ID,
		SourcePinID:       1,
		DestinationPinID:  2,
		ContentTypeID:     product.ID,
		Quantity:          10_000,
	})
	factory.CreateCharacterPlanet() // colony without pins

	newColonies := func(t *testing.T) *Colonies {
		return NewColonies(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
	}
	statusByPlanet := func(rows []colonyRow) map[int64]app.ColonyStatus {
		m := make(map[int64]app.ColonyStatus)
		for _, r := range rows {
			m[r.planetID] = r.status
		}
		return m
	}

	t.Run("should show forecasted status and work end", func(t *testing.T) {
		a := newColonies(t)
		rows, err := a.fetchRows(t.Context())
		require.NoError(t, err)
		require.Len(t, rows, 2)
		assert.Equal(t, app.ColonyExtracting, statusByPlanet(rows)[cp.EvePlanet.ID])
		for _, r := range rows {
			if r.planetID == cp.EvePlanet.ID {
				assert.True(t, r.workEndsAt.MustValue().Equal(expiry))
				assert.Equal(t, cp.EvePlanet.Type.IconID.ValueOrZero(), r.planetIconID)
			} else {
				assert.Equal(t, app.ColonyIdle, r.status)
				assert.True(t, r.workEndsAt.IsEmpty())
			}
		}
	})
	t.Run("should recalculate forecasts", func(t *testing.T) {
		a := newColonies(t)
		a.Update(t.Context())
		for i := range a.rows {
			a.rows[i].status = app.ColonyStatusUndefined
		}
		a.refreshForecasts()
		assert.Equal(t, app.ColonyExtracting, statusByPlanet(a.rows)[cp.EvePlanet.ID])
	})
	t.Run("should not invalidate a running update when refreshing", func(t *testing.T) {
		a := newColonies(t)
		isLatest := a.rowsRun.start() // simulates an update in flight
		a.refreshForecasts()
		assert.True(t, isLatest())
	})
	t.Run("should discard a refresh when update replaced the rows", func(t *testing.T) {
		var pending []func()
		orig := runAsync
		runAsync = func(f func()) { pending = append(pending, f) }
		t.Cleanup(func() { runAsync = orig })
		a := newColonies(t)
		a.refreshForecasts() // started with no rows
		a.Update(t.Context())
		for len(pending) > 0 {
			f := pending[0]
			pending = pending[1:]
			f()
		}
		assert.Len(t, a.rows, 2)
		assert.Len(t, a.rowsFiltered, 2)
	})
	t.Run("can filter by status on mobile", func(t *testing.T) {
		a := NewColonies(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: true,
			Storage:  st,
		}))
		a.Update(t.Context())
		require.Len(t, a.rowsFiltered, 2)
		a.filterChip.SetSelected(map[string]string{colonyFilterStatus: app.ColonyExtracting.Display()})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.Equal(t, cp.EvePlanet.ID, a.rowsFiltered[0].planetID)
		}
	})
	t.Run("can filter by status", func(t *testing.T) {
		a := newColonies(t)
		a.Update(t.Context())
		a.selectStatus.SetSelected(app.ColonyExtracting.Display())
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.Equal(t, cp.EvePlanet.ID, a.rowsFiltered[0].planetID)
		}
	})
}

func TestColonyRow_StatusShort(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name  string
		row   colonyRow
		want  string
		color fyne.ThemeColorName
	}{
		{
			"working shows remaining time",
			colonyRow{status: app.ColonyExtracting, workEndsAt: optional.New(now.Add(3 * time.Hour))},
			"3h 0m",
			"",
		},
		{
			"working beyond horizon",
			colonyRow{status: app.ColonyProducing, worksBeyond: true},
			colonyBeyondHorizonText,
			"",
		},
		{
			"working without end shows status",
			colonyRow{status: app.ColonyExtracting},
			app.ColonyExtracting.Display(),
			app.ColonyExtracting.Color(),
		},
		{
			"idle shows status",
			colonyRow{status: app.ColonyIdle},
			app.ColonyIdle.Display(),
			app.ColonyIdle.Color(),
		},
		{
			"needs attention shows status",
			colonyRow{status: app.ColonyNeedsAttention},
			app.ColonyNeedsAttention.Display(),
			app.ColonyNeedsAttention.Color(),
		},
		{
			"not setup shows status",
			colonyRow{status: app.ColonyNotSetup},
			app.ColonyNotSetup.Display(),
			app.ColonyNotSetup.Color(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.row.statusShort(now)
			require.Len(t, got, 1)
			seg := got[0].(*widget.TextSegment)
			assert.Equal(t, tc.want, seg.Text)
			assert.Equal(t, tc.color, seg.Style.ColorName)
		})
	}
}

func TestColonyListItem(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should show attention icon only for colonies with problems", func(t *testing.T) {
		w := newColonyListItem()
		win := test.NewWindow(w)
		t.Cleanup(win.Close)
		w.set(colonyRow{status: app.ColonyNeedsAttention})
		assert.True(t, w.planet.attention.Visible())
		w.set(colonyRow{status: app.ColonyExtracting}) // row is recycled
		assert.False(t, w.planet.attention.Visible())
		w.set(colonyRow{status: app.ColonyIdle})
		assert.False(t, w.planet.attention.Visible())
	})
}

func TestColonyFilter_Match(t *testing.T) {
	r := colonyRow{
		extracting:      set.Of("Aqueous Liquids"),
		ownerName:       "Bruce",
		planetTypeName:  "Barren",
		producing:       set.Of("Water"),
		regionName:      "The Forge",
		solarSystemName: "Jita",
		status:          app.ColonyExtracting,
		tags:            set.Of("Main"),
	}
	for _, tc := range []struct {
		name   string
		filter colonyFilter
		want   bool
	}{
		{"no filter", colonyFilter{}, true},
		{"all filters match", colonyFilter{
			extracted:   "Aqueous Liquids",
			owner:       "Bruce",
			planetType:  "Barren",
			produced:    "Water",
			region:      "The Forge",
			solarSystem: "Jita",
			status:      app.ColonyExtracting.Display(),
			tag:         "Main",
		}, true},
		{"attention", colonyFilter{attention: true}, false},
		{"extracted", colonyFilter{extracted: "Base Metals"}, false},
		{"owner", colonyFilter{owner: "Alice"}, false},
		{"planet type", colonyFilter{planetType: "Lava"}, false},
		{"produced", colonyFilter{produced: "Oxygen"}, false},
		{"region", colonyFilter{region: "Domain"}, false},
		{"solar system", colonyFilter{solarSystem: "Amarr"}, false},
		{"status", colonyFilter{status: app.ColonyIdle.Display()}, false},
		{"tag", colonyFilter{tag: "Alt"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(r))
		})
	}
	t.Run("attention matches colonies with problems", func(t *testing.T) {
		for _, s := range []app.ColonyStatus{app.ColonyNeedsAttention, app.ColonyNotSetup} {
			assert.True(t, colonyFilter{attention: true}.match(colonyRow{status: s}), s.Display())
		}
		assert.False(t, colonyFilter{attention: true}.match(colonyRow{status: app.ColonyIdle}))
	})
}

func TestColonyRow_SetForecast(t *testing.T) {
	const iconID = 1047
	t.Run("should cache grayscale planet icon for problems", func(t *testing.T) {
		grayscalePlanetIconCache.Delete(iconID)
		r := colonyRow{planetIconID: iconID}
		r.setForecast(&app.ColonyForecast{Status: app.ColonyNeedsAttention})
		_, ok := grayscalePlanetIconCache.Load(iconID)
		assert.True(t, ok)
	})
	t.Run("should not cache grayscale planet icon when working", func(t *testing.T) {
		grayscalePlanetIconCache.Delete(iconID)
		r := colonyRow{planetIconID: iconID}
		r.setForecast(&app.ColonyForecast{Status: app.ColonyExtracting})
		_, ok := grayscalePlanetIconCache.Load(iconID)
		assert.False(t, ok)
	})
}
