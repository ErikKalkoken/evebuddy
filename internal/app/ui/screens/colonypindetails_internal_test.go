package screens

import (
	"context"
	"fmt"
	"slices"
	"strings"
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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestColonyPinDetails(t *testing.T) {
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
	aqueousLiquids := factory.CreateEveType(storage.CreateEveTypeParams{ID: 2268, Name: "Aqueous Liquids", Volume: optional.New(0.01)})
	schematic := factory.CreateEveSchematic(storage.CreateEveSchematicParams{ID: 121, Name: "Water"})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 3645, Name: "Water"}) // output of Water
	biofuels := factory.CreateEveSchematic(storage.CreateEveSchematicParams{ID: 134, Name: "Biofuels"})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 2288, Name: "Carbon Compounds"})              // input of Biofuels, not referenced by colony
	proteins := factory.CreateEveSchematic(storage.CreateEveSchematicParams{ID: 135, Name: "Proteins"}) // types unknown
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID:      cp.ID,
		PinID:                  1,
		TypeID:                 ecuType.ID,
		ExtractorProductTypeID: optional.New(aqueousLiquids.ID),
		ExtractorQtyPerCycle:   optional.New[int64](1081),
		ExtractorCycleTime:     optional.New(30 * time.Minute),
		ExtractorNumHeads:      optional.New[int64](10),
		InstallTime:            optional.New(lastUpdate),
		ExpiryTime:             optional.New(lastUpdate.Add(4 * time.Hour)),
		LastCycleStart:         optional.New(lastUpdate),
	})
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID: cp.ID,
		PinID:             2,
		TypeID:            storageType.ID,
		Contents:          map[int64]int64{aqueousLiquids.ID: 100_000},
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
		TypeID:            processorType.ID,
		SchematicID:       optional.New(biofuels.ID),
	})
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID: cp.ID,
		PinID:             5,
		TypeID:            processorType.ID,
		SchematicID:       optional.New(proteins.ID),
	})
	factory.CreatePlanetPin(storage.CreatePlanetPinParams{
		CharacterPlanetID: cp.ID,
		PinID:             6,
		TypeID:            processorType.ID, // without schematic
	})
	factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{
		CharacterPlanetID: cp.ID,
		RouteID:           1,
		SourcePinID:       1,
		DestinationPinID:  2,
		ContentTypeID:     aqueousLiquids.ID,
		Quantity:          10_000,
	})
	factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{
		CharacterPlanetID: cp.ID,
		RouteID:           2,
		SourcePinID:       2,
		DestinationPinID:  3,
		ContentTypeID:     aqueousLiquids.ID,
		Quantity:          3000,
	})

	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	type shownPin struct {
		pinID int64
		title string
	}
	var shownPins []shownPin
	makeInfo := func(t *testing.T, pinID int64) colonyPinInfo {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, pinID, func(pinID int64, title string) {
			shownPins = append(shownPins, shownPin{pinID, title})
		})
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		f := u.Character().ForecastPlanet(a.colony, now)
		return a.makeInfo(a.colony, f, a.extraTypeNames, now)
	}
	item := func(t *testing.T, items []ui.AttributeItem, label string) ui.AttributeItem {
		for _, x := range items {
			if x.Label == label {
				return x
			}
		}
		t.Fatalf("item not found: %s", label)
		return ui.AttributeItem{}
	}
	hasItem := func(items []ui.AttributeItem, label string) bool {
		return slices.ContainsFunc(items, func(x ui.AttributeItem) bool { return x.Label == label })
	}
	value := func(t *testing.T, items []ui.AttributeItem, label string) string {
		return item(t, items, label).Value
	}
	// lines returns all items as "label value" lines.
	lines := func(items []ui.AttributeItem) []string {
		var s []string
		for _, x := range items {
			s = append(s, strings.TrimSpace(x.Label+" "+x.Value))
		}
		return s
	}
	statusText := func(info colonyPinInfo) string {
		var s string
		for _, x := range info.status {
			s += x.Textual()
		}
		return s
	}

	t.Run("should show extractor", func(t *testing.T) {
		info := makeInfo(t, 1)
		require.True(t, info.found)
		assert.Equal(t, "Extractor 21-111", info.name) // pin ID 1
		assert.Equal(t, "Aqueous Liquids", info.product)
		assert.NotNil(t, info.onProduct)
		assert.Equal(t, pinTypeExtractor.icon(), info.symbolIcon)
		assert.Equal(t, pinTypeExtractor, info.symbolType)
		assert.Equal(t, app.PinExtracting.Display(), statusText(info))
		assert.True(t, info.progress.ValueOrZero() > 0)
		assert.Equal(t, "Aqueous Liquids", value(t, info.main, "Product"))
		assert.NotNil(t, item(t, info.main, "Product").InfoAction)
		assert.Contains(t, value(t, info.main, "Expires"), "(in ")
		assert.Equal(t, "10", value(t, info.main, "Heads"))
		assert.Equal(t, "1,081", value(t, info.main, "Base yield"))
		assert.True(t, hasItem(info.main, "Data from"))
		assert.Nil(t, info.storage)
		assert.Nil(t, info.inputs)
		assert.Equal(t, []string{"Outgoing", "Aqueous Liquids x 10,000 Storage 31-111"}, lines(info.routes))
		assert.True(t, info.routes[0].IsHeading)
		shownPins = nil
		info.routes[1].InfoAction()
		assert.Equal(t, []shownPin{{2, "Storage 31-111 on " + cp.EvePlanet.Name}}, shownPins, "opens connected installation")
	})
	t.Run("should show extractor program", func(t *testing.T) {
		info := makeInfo(t, 1)
		require.Len(t, info.program, 8) // 4 hours with 30 minute cycles
		var phases []colonyCyclePhase
		for _, c := range info.program {
			phases = append(phases, c.phase)
		}
		assert.Equal(t, []colonyCyclePhase{
			cycleCompleted, cycleCompleted, cycleCurrent, cycleUpcoming,
			cycleUpcoming, cycleUpcoming, cycleUpcoming, cycleUpcoming,
		}, phases)
		assert.Equal(t, lastUpdate, info.program[0].start)
		assert.Equal(t, lastUpdate.Add(30*time.Minute), info.program[1].start)
		assert.Positive(t, info.program[0].output)
		assert.Contains(t, info.programTitle, "Total ")
		assert.Contains(t, info.programTitle, "Current ")
		assert.False(t, hasItem(info.main, "Total output"), "only shown in program")
	})
	t.Run("should show program chart", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 1, nil)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		assert.Contains(t, a.programTitle.Text, "Total ")
		assert.Len(t, a.programLegend.container.Objects, 3)
	})
	t.Run("should show processor", func(t *testing.T) {
		info := makeInfo(t, 3)
		require.True(t, info.found)
		assert.Equal(t, "Water", info.product)
		assert.Equal(t, "Water x 20", value(t, info.main, "Schematic"))
		assert.NotNil(t, item(t, info.main, "Schematic").InfoAction)
		assert.Nil(t, info.storage)
		require.Len(t, info.inputs, 1)
		x := info.inputs[0]
		assert.Equal(t, aqueousLiquids.ID, x.typeID)
		assert.Equal(t, "Aqueous Liquids", x.name)
		assert.EqualValues(t, 3_000, x.demand)
		assert.True(t, x.isRouted)
		assert.Nil(t, info.program)
		assert.Equal(t, []string{"Incoming", "Aqueous Liquids x 3,000 Storage 31-111"}, lines(info.routes))
	})
	t.Run("should show processor without schematic", func(t *testing.T) {
		info := makeInfo(t, 6)
		require.True(t, info.found)
		assert.Empty(t, info.product)
		assert.Equal(t, "-", value(t, info.main, "Schematic"))
		assert.Empty(t, info.inputs)
		assert.NotNil(t, info.inputs, "shows inputs tab")
		assert.Equal(t, "No schematic", info.inputsEmpty)
	})
	t.Run("should show last activity of producing processor", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 3, nil)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		start := now.Add(-20 * time.Minute)
		f := &app.ColonyForecast{Pins: map[int64]*app.PinForecast{3: {
			IsActive:       true,
			LastCycleStart: optional.New(start),
			LastRunTime:    optional.New(start),
		}}}
		info := a.makeInfo(a.colony, f, a.extraTypeNames, now)
		assert.Equal(t, start.Format(app.DateTimeFormat), value(t, info.main, "Last activity"))
		assert.False(t, hasItem(info.main, "Idle for"))
	})
	t.Run("should show last production of idle processor", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 3, nil)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		start := now.Add(-3 * time.Hour) // cycle time is 1 hour
		f := &app.ColonyForecast{Pins: map[int64]*app.PinForecast{3: {
			LastCycleStart: optional.New(start),
			LastRunTime:    optional.New(now.Add(-10 * time.Minute)), // last check for inputs
		}}}
		info := a.makeInfo(a.colony, f, a.extraTypeNames, now)
		assert.Equal(t, start.Add(time.Hour).Format(app.DateTimeFormat), value(t, info.main, "Last activity"))
		assert.Equal(t, ihumanize.Duration(2*time.Hour), value(t, info.main, "Idle for"))
	})
	t.Run("should show unknown activity of processor that never ran", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 3, nil)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		f := &app.ColonyForecast{Pins: map[int64]*app.PinForecast{3: {
			LastRunTime: optional.New(now.Add(-10 * time.Minute)),
		}}}
		info := a.makeInfo(a.colony, f, a.extraTypeNames, now)
		assert.Equal(t, "-", value(t, info.main, "Last activity"))
		assert.False(t, hasItem(info.main, "Idle for"))
	})
	t.Run("should show name of unreferenced input from database", func(t *testing.T) {
		info := makeInfo(t, 4)
		require.True(t, info.found)
		assert.Equal(t, []colonyInputItem{{demand: 3_000, name: "Carbon Compounds", typeID: 2288}}, info.inputs)
		assert.Equal(t, "Biofuels x 20", value(t, info.main, "Schematic"))
	})
	t.Run("should show fallback for input missing in database", func(t *testing.T) {
		info := makeInfo(t, 5)
		require.True(t, info.found)
		assert.Equal(t, []colonyInputItem{{demand: 3_000, name: "Type #2305", typeID: 2305}}, info.inputs)
	})
	t.Run("should show storage", func(t *testing.T) {
		info := makeInfo(t, 2)
		require.True(t, info.found)
		assert.Empty(t, info.product)
		assert.Equal(t, "-", statusText(info))
		assert.Contains(t, value(t, info.main, "Capacity"), " / 12,000 m3")
		assert.Nil(t, info.inputs)
		require.Len(t, info.storage, 1)
		x := info.storage[0]
		assert.Equal(t, aqueousLiquids.ID, x.typeID)
		assert.Equal(t, "Aqueous Liquids", x.name)
		assert.Equal(t, aqueousLiquids.Group.Name, x.group)
		assert.Positive(t, x.quantity)
		assert.InDelta(t, float64(x.quantity)*0.01, x.volume, 0.0001)
		assert.Equal(t, []string{
			"Incoming",
			"Aqueous Liquids x 10,000 Extractor 21-111",
			"Outgoing",
			"Aqueous Liquids x 3,000 Basic Processor 41-111",
		}, lines(info.routes))
	})
	t.Run("should show tabs for installation type", func(t *testing.T) {
		titles := func(a *colonyPinDetails) []string {
			var s []string
			for _, x := range a.tabs.Items {
				s = append(s, x.Text)
			}
			return s
		}
		for pinID, want := range map[int64][]string{
			1: {"Main", "Program", "Routes"},
			2: {"Main", "Storage", "Routes"},
			3: {"Main", "Inputs", "Routes"},
			6: {"Main", "Inputs", "Routes"},
		} {
			a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, pinID, nil)
			t.Cleanup(a.stop)
			require.NoError(t, a.Update(t.Context()))
			assert.Equal(t, want, titles(a), "pin %d", pinID)
		}
	})
	t.Run("should report missing installation", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 99, nil)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		require.Len(t, a.content.Objects, 1)
		assert.Equal(t, "Installation no longer exists", a.content.Objects[0].(*widget.Label).Text)
	})
	t.Run("should clear issue after successful update", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 1, nil)
		t.Cleanup(a.stop)
		a.setIssue("ERROR: failed")
		require.NoError(t, a.Update(t.Context()))
		assert.Empty(t, a.footer.Text)
		assert.Equal(t, widget.MediumImportance, a.footer.Importance)
	})
	t.Run("should stop forecasting when colony no longer exists", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 1, nil)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		a.planetID = 42 // a colony which does not exist
		require.NoError(t, a.Update(t.Context()))
		assert.Nil(t, a.colony)
		require.Len(t, a.content.Objects, 1)
		assert.Equal(t, "Installation no longer exists", a.content.Objects[0].(*widget.Label).Text)
		a.content.Objects = nil
		a.refreshForecast()
		assert.Empty(t, a.content.Objects)
	})
	t.Run("should stop forecasting when colony can not be loaded", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 1, nil)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.Error(t, a.Update(ctx))
		assert.Nil(t, a.colony)
	})
	t.Run("should recalculate forecast", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 1, nil)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		a.content.Objects = nil
		a.refreshForecast()
		assert.NotEmpty(t, a.content.Objects)
	})
	t.Run("should stop forecasting when character is removed", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 1, nil)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		u.Signals().CharacterRemoved.Emit(t.Context(), &app.EntityShort{ID: character.ID})
		assert.Eventually(t, func() bool {
			var removed bool
			fyne.DoAndWait(func() { removed = a.colony == nil })
			return removed
		}, time.Second, 10*time.Millisecond)
		a.content.Objects = nil
		a.refreshForecast()
		assert.Empty(t, a.content.Objects)
	})
}

func TestGroupCycles(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	makeCycles := func(phases ...colonyCyclePhase) []colonyCycle {
		var s []colonyCycle
		for i, p := range phases {
			s = append(s, colonyCycle{index: i, output: int64(i + 1), phase: p, start: start.Add(time.Duration(i) * time.Hour)})
		}
		return s
	}
	t.Run("should keep cycles when below maximum", func(t *testing.T) {
		cycles := makeCycles(cycleCompleted, cycleCurrent, cycleUpcoming)
		assert.Equal(t, cycles, groupCycles(cycles, 3))
	})
	t.Run("should merge cycles into steps with average output", func(t *testing.T) {
		cycles := makeCycles(cycleUpcoming, cycleUpcoming, cycleUpcoming, cycleUpcoming)
		assert.Equal(t, []colonyCycle{
			{index: 0, output: 2, phase: cycleUpcoming, start: start},                    // (1+2)/2 rounded
			{index: 1, output: 4, phase: cycleUpcoming, start: start.Add(2 * time.Hour)}, // (3+4)/2 rounded
		}, groupCycles(cycles, 2))
	})
	t.Run("should not merge cycles of different phases", func(t *testing.T) {
		cycles := makeCycles(cycleCompleted, cycleCompleted, cycleCompleted, cycleCurrent, cycleUpcoming, cycleUpcoming)
		got := groupCycles(cycles, 3)
		var phases []colonyCyclePhase
		var starts []time.Time
		for i, c := range got {
			assert.Equal(t, i, c.index)
			phases = append(phases, c.phase)
			starts = append(starts, c.start)
		}
		assert.Equal(t, []colonyCyclePhase{cycleCompleted, cycleCompleted, cycleCurrent, cycleUpcoming}, phases)
		assert.Equal(t, []time.Time{start, start.Add(2 * time.Hour), start.Add(3 * time.Hour), start.Add(4 * time.Hour)}, starts)
	})
	t.Run("should limit steps for long programs", func(t *testing.T) {
		phases := make([]colonyCyclePhase, 500)
		for i := range phases {
			switch {
			case i < 200:
				phases[i] = cycleCompleted
			case i == 200:
				phases[i] = cycleCurrent
			default:
				phases[i] = cycleUpcoming
			}
		}
		got := groupCycles(makeCycles(phases...), colonyProgramMaxSteps)
		assert.LessOrEqual(t, len(got), colonyProgramMaxSteps+3)
	})
}

func TestSortByNameAndQuantity(t *testing.T) {
	item := func(name string, quantity int64, value string) quantityItem {
		return quantityItem{name: name, quantity: quantity, item: ui.AttributeItem{Label: fmt.Sprintf("%s x %d", name, quantity), Value: value}}
	}
	got := sortByNameAndQuantity([]quantityItem{
		item("Water", 20, "A"),
		item("Base Metals", 3_000, "B"),
		item("Base Metals", 10_000, "C"),
		item("Base Metals", 3_000, "A"),
	})
	var labels []string
	for _, x := range got {
		labels = append(labels, x.Label+" "+x.Value)
	}
	assert.Equal(t, []string{
		"Base Metals x 10000 C",
		"Base Metals x 3000 A",
		"Base Metals x 3000 B",
		"Water x 20 A",
	}, labels)
}
