package screens

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestColonyDetails(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	now := time.Now().UTC()
	lastUpdate := now.Add(-65 * time.Minute)
	character := factory.CreateCharacterFull()
	cp := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
		CharacterID: character.ID,
		LastUpdate:  lastUpdate,
	})
	prefix := cp.EvePlanet.TypeDisplay() + " "
	ecuGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupExtractorControlUnits})
	ecuType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: ecuGroup.ID, Name: prefix + "Extractor Control Unit"})
	storageGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupStorageFacilities})
	storageType := factory.CreateEveType(storage.CreateEveTypeParams{
		GroupID:  storageGroup.ID,
		Name:     prefix + "Storage Facility",
		Capacity: optional.New(12_000.0),
	})
	commandCenterGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupCommandCenters})
	commandCenterType := factory.CreateEveType(storage.CreateEveTypeParams{
		GroupID:  commandCenterGroup.ID,
		Name:     prefix + "Command Center",
		Capacity: optional.New(500.0),
	})
	processorGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupProcessors})
	processorType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: processorGroup.ID, Name: prefix + "Basic Industry Facility"})
	product := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Base Metals", Volume: optional.New(0.01)})
	schematic := factory.CreateEveSchematic(storage.CreateEveSchematicParams{ID: 121, Name: "Water"})
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
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID: cp.ID,
		PinID:             3,
		TypeID:            processorType.ID,
		SchematicID:       optional.New(schematic.ID),
	})
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID: cp.ID,
		PinID:             4,
		TypeID:            commandCenterType.ID,
	})
	factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{
		CharacterPlanetID: cp.ID,
		SourcePinID:       1,
		DestinationPinID:  2,
		ContentTypeID:     product.ID,
		Quantity:          10_000,
	})

	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	a := newColonyDetails(u, character.ID, cp.EvePlanet.ID)
	err := a.Update(t.Context())
	require.NoError(t, err)

	rowByType := func(t *testing.T, pt colonyPinType) colonyDetailsRow {
		for _, r := range a.rows {
			if r.pinType == pt {
				return r
			}
		}
		t.Fatalf("row not found: %s", pt)
		return colonyDetailsRow{}
	}

	t.Run("should lay out header without gaps after width changed", func(t *testing.T) {
		a := newColonyDetails(u, character.ID, cp.EvePlanet.ID)
		t.Cleanup(a.stop)
		w := test.NewWindow(a)
		t.Cleanup(w.Close)
		w.Resize(fyne.NewSize(160, 640)) // narrow, so the header texts wrap
		require.NoError(t, a.Update(t.Context()))
		w.Resize(fyne.NewSize(360, 640))
		for _, o := range []fyne.CanvasObject{a.planet, a.planetType, a.owner, a.status} {
			assert.Equal(t, o.MinSize().Height, o.Size().Height)
		}
	})
	t.Run("should show colony status", func(t *testing.T) {
		// factory without input route makes the colony not setup
		assert.Contains(t, a.status.String(), app.ColonyNotSetup.Display())
	})
	t.Run("should name pins with designator", func(t *testing.T) {
		r := rowByType(t, pinTypeExtractor)
		assert.Equal(t, "Extractor 21-111", r.name) // pin ID 1
		assert.Contains(t, r.searchTarget, "21-111")
	})
	t.Run("should show extractor with remaining time", func(t *testing.T) {
		r := rowByType(t, pinTypeExtractor)
		assert.Equal(t, "Base Metals", r.output)
		assert.Equal(t, expiry.Format(app.DateTimeFormat), r.info)
		assert.NotEqual(t, app.PinExtracting.Display(), segmentsText(r.status))
		assert.InDelta(t, 65.0/240.0, r.progress.MustValue(), 0.01, "elapsed share of the program")
	})
	t.Run("should show storage contents and fill", func(t *testing.T) {
		r := rowByType(t, pinTypeStorage)
		assert.Equal(t, "Base Metals", r.output)
		assert.Equal(t, "46 / 12,000 m3", r.info)
		assert.Equal(t, "0%", segmentsText(r.status))
		assert.InDelta(t, 4553*0.01/12_000, r.progress.MustValue(), 0.0001, "fill level")
	})
	t.Run("should show command center like storage without level", func(t *testing.T) {
		r := rowByType(t, pinTypeCommandCenter)
		assert.Equal(t, "-", r.output)
		assert.Equal(t, "0 / 500 m3", r.info)
	})
	t.Run("should show factory status", func(t *testing.T) {
		r := rowByType(t, pinTypeBasicProcessor)
		assert.Equal(t, "Water", r.output)
		assert.Equal(t, app.PinInputNotRouted.Display(), segmentsText(r.status))
		assert.True(t, r.progress.IsEmpty(), "not producing")
	})
	t.Run("should show factory with schematic only in factory details", func(t *testing.T) {
		cp2 := *a.colony
		cp2.Pins = nil
		for _, p := range a.colony.Pins {
			if p.Type.Group.ID == app.EveGroupProcessors {
				p2 := *p
				p2.FactorySchematic, p2.Schematic = p.Schematic, optional.Optional[*app.EveSchematic]{}
				p = &p2
			}
			cp2.Pins = append(cp2.Pins, p)
		}
		_, _, rows := a.makeRows(&cp2, time.Now())
		var found bool
		for _, r := range rows {
			if r.pinType == pinTypeBasicProcessor {
				found = true
				assert.Equal(t, "Water", r.output)
			}
		}
		assert.True(t, found)
	})
	t.Run("should recalculate forecast", func(t *testing.T) {
		a.rows = nil
		a.refreshForecast()
		assert.Len(t, a.rows, 4)
	})
	t.Run("should show installation when selected", func(t *testing.T) {
		var pinID int64
		var title string
		a.showPin = func(id int64, s string) {
			pinID, title = id, s
		}
		defer func() { a.showPin = nil }()
		i := slices.IndexFunc(a.rowsFiltered, func(r colonyDetailsRow) bool { return r.pinType == pinTypeStorage })
		require.NotEqual(t, -1, i)
		a.installations.Select(i)
		assert.EqualValues(t, 2, pinID)
		assert.Equal(t, "Storage 31-111 on "+cp.EvePlanet.Name, title) // pin ID 2
	})
	t.Run("should navigate between colony and installations in one window", func(t *testing.T) {
		r := colonyRow{characterID: character.ID, planetID: cp.EvePlanet.ID, planetName: cp.EvePlanet.Name}
		// listeners returns the number of listeners of the signals each page listens to.
		listeners := func() []int {
			sig := u.Signals()
			return []int{sig.RefreshTickerExpired.Len(), sig.CharacterSectionChanged.Len(), sig.CharacterRemoved.Len()}
		}
		base := listeners()
		assertPages := func(t *testing.T, n int, msg string) {
			t.Helper()
			var want []int
			for _, x := range base {
				want = append(want, x+n)
			}
			assert.Equal(t, want, listeners(), "listening pages %s", msg)
		}
		showColonyDetailsWindow(u, r)
		w, created, _ := u.GetOrCreateWindowWithOnClosed(fmt.Sprintf("colony-%d-%d", character.ID, cp.EvePlanet.ID))
		require.False(t, created)
		nav := w.Content().(*xwidget.Navigator)
		title := func() string {
			return nav.Current().(*xwidget.AppBar).Title()
		}
		root := title()

		var details *colonyDetails
		for _, o := range test.LaidOutObjects(w.Content()) {
			if x, ok := o.(*colonyDetails); ok {
				details = x
			}
		}
		require.NotNil(t, details)
		assertPages(t, 1, "after open")
		details.showPin(1, "Extractor")
		assert.Equal(t, "Extractor", title())

		var pin *colonyPinDetails
		for _, o := range test.LaidOutObjects(nav.Current()) {
			if x, ok := o.(*colonyPinDetails); ok {
				pin = x
			}
		}
		require.NotNil(t, pin)
		pin.showPin(2, "Storage")
		assert.Equal(t, "Storage", title())
		assertPages(t, 3, "with two pins")

		nav.Pop()
		assert.Equal(t, "Extractor", title())
		assertPages(t, 2, "after back")

		showColonyDetailsWindow(u, r) // reopening shows the colony again
		assert.Equal(t, root, title())
		assert.True(t, nav.IsRoot())
		assertPages(t, 1, "after reopening")

		details.showPin(1, "Extractor")
		w.Close()
		assertPages(t, 0, "after close")
	})
	t.Run("should update planet icon on refresh", func(t *testing.T) {
		a.icon.icon.Resource = nil
		a.refreshForecast()
		want := colonyPlanetIcon(cp.EvePlanet.Type.IconID.ValueOrZero(), true) // colony is not setup
		assert.Equal(t, want, a.icon.icon.Resource)
		assert.True(t, a.icon.attention.Visible())
	})
	t.Run("should not restore colony on refresh after a failed update", func(t *testing.T) {
		a.characterID.Store(0)
		defer a.characterID.Store(character.ID)
		require.NoError(t, a.Update(t.Context()))
		a.refreshForecast()
		assert.Empty(t, a.rows)
	})
	t.Run("should stop forecasting when character is removed", func(t *testing.T) {
		require.NoError(t, a.Update(t.Context()))
		u.Signals().CharacterRemoved.Emit(t.Context(), &app.EntityShort{ID: character.ID})
		assert.Eventually(t, func() bool {
			var removed bool
			fyne.DoAndWait(func() { removed = a.colony == nil })
			return removed
		}, time.Second, 10*time.Millisecond)
		a.rows = nil
		a.refreshForecast()
		assert.Empty(t, a.rows)
	})
}

func TestColonyProgress(t *testing.T) {
	cases := []struct {
		name           string
		elapsed, total time.Duration
		want           float64
	}{
		{"should return ratio", 15 * time.Minute, time.Hour, 0.25},
		{"should clamp to 1", 2 * time.Hour, time.Hour, 1},
		{"should clamp to 0", -time.Minute, time.Hour, 0},
		{"should return 0 for no total", time.Minute, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.want, colonyProgress(tc.elapsed, tc.total), 0.0001)
		})
	}
}

func TestColonyContentsDisplay(t *testing.T) {
	names := map[int64]string{1: "Alpha", 2: "Bravo", 3: "Charlie", 4: "Delta"}
	volumes := map[int64]float64{1: 1, 2: 1, 3: 1, 4: 100}
	cases := []struct {
		name     string
		contents map[int64]int64
		want     string
	}{
		{"empty", map[int64]int64{}, ""},
		{"single", map[int64]int64{1: 5}, "Alpha"},
		{"largest first", map[int64]int64{1: 5, 2: 1_000}, "Bravo, +1 more"},
		{"limited", map[int64]int64{1: 4, 2: 3, 3: 2}, "Alpha, +2 more"},
		{"largest volume first", map[int64]int64{1: 50, 4: 1}, "Delta, +1 more"},
		{"tie in volume sorted by amount", map[int64]int64{98: 1, 99: 5}, "Type #99, +1 more"},
		{"tie sorted by name", map[int64]int64{2: 7, 1: 7}, "Alpha, +1 more"},
		{"unknown type", map[int64]int64{99: 1}, "Type #99"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, colonyContentsDisplay(tc.contents, names, volumes))
		})
	}
}

func segmentsText(segments []widget.RichTextSegment) string {
	var s string
	for _, x := range segments {
		s += x.Textual()
	}
	return s
}
