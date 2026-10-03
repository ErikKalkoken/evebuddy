package screens

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestColonyDetails(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
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

	rowByName := func(t *testing.T, name string) colonyDetailsRow {
		for _, r := range a.rows {
			if r.name == name {
				return r
			}
		}
		t.Fatalf("row not found: %s", name)
		return colonyDetailsRow{}
	}

	t.Run("should show colony status", func(t *testing.T) {
		// factory without input route makes the colony not setup
		assert.Contains(t, a.status.String(), app.ColonyNotSetup.Display())
	})
	t.Run("should show extractor with remaining time", func(t *testing.T) {
		r := rowByName(t, string(pinTypeExtractor))
		assert.Equal(t, "Base Metals", r.output)
		assert.Equal(t, expiry.Format(app.DateTimeFormat), r.info)
		assert.NotEqual(t, app.PinExtracting.Display(), segmentsText(r.status))
		assert.InDelta(t, 65.0/240.0, r.progress.MustValue(), 0.01, "elapsed share of the program")
	})
	t.Run("should show storage contents and fill", func(t *testing.T) {
		r := rowByName(t, string(pinTypeStorage))
		assert.Equal(t, "Base Metals 4,553", r.output)
		assert.Equal(t, "46 / 12,000 m3", r.info)
		assert.Equal(t, "0%", segmentsText(r.status))
		assert.InDelta(t, 4553*0.01/12_000, r.progress.MustValue(), 0.0001, "fill level")
	})
	t.Run("should show factory status", func(t *testing.T) {
		r := rowByName(t, string(pinTypeBasicProcessor))
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
			if r.name == string(pinTypeBasicProcessor) {
				found = true
				assert.Equal(t, "Water", r.output)
			}
		}
		assert.True(t, found)
	})
	t.Run("should recalculate forecast", func(t *testing.T) {
		a.rows = nil
		a.refreshForecast()
		assert.Len(t, a.rows, 3)
	})
	t.Run("should show installation when selected", func(t *testing.T) {
		var pinID int64
		var title string
		a.showPin = func(id int64, s string) {
			pinID, title = id, s
		}
		defer func() { a.showPin = nil }()
		i := slices.IndexFunc(a.rowsFiltered, func(r colonyDetailsRow) bool { return r.name == string(pinTypeStorage) })
		require.NotEqual(t, -1, i)
		a.installations.Select(i)
		assert.EqualValues(t, 2, pinID)
		assert.Equal(t, string(pinTypeStorage)+" on "+cp.EvePlanet.Name, title)
	})
	t.Run("should navigate between colony and installations in one window", func(t *testing.T) {
		r := colonyRow{characterID: character.ID, planetID: cp.EvePlanet.ID, planetName: cp.EvePlanet.Name}
		showColonyDetailsWindow(u, r)
		w, created, _ := u.GetOrCreateWindowWithOnClosed(fmt.Sprintf("colony-%d-%d", character.ID, cp.EvePlanet.ID))
		require.False(t, created)
		defer w.Close()
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

		nav.Pop()
		assert.Equal(t, "Extractor", title())

		showColonyDetailsWindow(u, r) // reopening shows the colony again
		assert.Equal(t, root, title())
		assert.True(t, nav.IsRoot())
	})
	t.Run("should update planet icon on refresh", func(t *testing.T) {
		a.icon.Resource = nil
		a.refreshForecast()
		want := colonyPlanetIcon(cp.EvePlanet.Type.IconID.ValueOrZero(), true) // colony is not setup
		assert.Equal(t, want, a.icon.Resource)
	})
	t.Run("should not restore colony on refresh after a failed update", func(t *testing.T) {
		a.characterID.Store(0)
		defer a.characterID.Store(character.ID)
		require.NoError(t, a.Update(t.Context()))
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
	cases := []struct {
		name     string
		contents map[int64]int64
		want     string
	}{
		{"empty", map[int64]int64{}, ""},
		{"sorted by amount", map[int64]int64{1: 5, 2: 1_000}, "Bravo 1,000, Alpha 5"},
		{"limited", map[int64]int64{1: 4, 2: 3, 3: 2, 4: 1}, "Alpha 4, Bravo 3, Charlie 2, +1 more"},
		{"unknown type", map[int64]int64{99: 1}, "Type #99 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, colonyContentsDisplay(tc.contents, names))
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
