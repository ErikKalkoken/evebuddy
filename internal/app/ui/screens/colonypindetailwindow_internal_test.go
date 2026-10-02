package screens

import (
	"slices"
	"strings"
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
	makeInfo := func(t *testing.T, pinID int64) colonyPinInfo {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, pinID)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		f := u.Character().ForecastPlanet(a.colony, now)
		return a.makeInfo(a.colony, f, a.ownerName, a.extraTypeNames, now)
	}
	field := func(t *testing.T, fields []colonyPinField, label string) colonyPinField {
		for _, x := range fields {
			if x.label == label {
				return x
			}
		}
		t.Fatalf("field not found: %s", label)
		return colonyPinField{}
	}
	hasField := func(fields []colonyPinField, label string) bool {
		return slices.ContainsFunc(fields, func(x colonyPinField) bool { return x.label == label })
	}
	// value returns the text of a field, with one line per item line.
	value := func(t *testing.T, fields []colonyPinField, label string) string {
		for _, x := range fields {
			if x.label != label {
				continue
			}
			if len(x.lines) == 0 {
				return x.value
			}
			var lines []string
			for _, l := range x.lines {
				lines = append(lines, l.text())
			}
			return strings.Join(lines, "\n")
		}
		t.Fatalf("field not found: %s", label)
		return ""
	}

	t.Run("should show extractor", func(t *testing.T) {
		info := makeInfo(t, 1)
		require.True(t, info.found)
		assert.Equal(t, string(pinTypeExtractor), value(t, info.general, "Installation"))
		assert.NotNil(t, field(t, info.general, "Installation").icon)
		assert.Equal(t, cp.EvePlanet.Name, value(t, info.general, "Colony"))
		assert.Equal(t, character.EveCharacter.Name, value(t, info.general, "Owner"))
		assert.Equal(t, app.PinExtracting.Display(), value(t, info.general, "Status"))
		assert.Equal(t, "Aqueous Liquids", value(t, info.specific, "Product"))
		assert.Contains(t, value(t, info.specific, "Expires"), "(in ")
		assert.Equal(t, "10", value(t, info.specific, "Heads"))
		assert.Equal(t, "1,081", value(t, info.specific, "Base quantity per cycle"))
		assert.Equal(t, "None", value(t, info.routes, "Incoming routes"))
		assert.Equal(t, "Aqueous Liquids x 10,000 to Storage", value(t, info.routes, "Outgoing routes"))
	})
	t.Run("should show processor", func(t *testing.T) {
		info := makeInfo(t, 3)
		require.True(t, info.found)
		assert.Equal(t, "Water x 20", value(t, info.specific, "Schematic"))
		inputs := value(t, info.specific, "Inputs")
		assert.Contains(t, inputs, "Aqueous Liquids ")
		assert.Contains(t, inputs, " / 3,000")
		assert.NotContains(t, inputs, "not routed")
		assert.Equal(t, int64(3645), field(t, info.specific, "Schematic").lines[0].typeID, "output links to its type")
		assert.Equal(t, aqueousLiquids.ID, field(t, info.specific, "Inputs").lines[0].typeID)
		assert.Equal(t, "Aqueous Liquids x 3,000 from Storage", value(t, info.routes, "Incoming routes"))
	})
	t.Run("should show last activity of producing processor", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 3)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		start := now.Add(-20 * time.Minute)
		f := &app.ColonyForecast{Pins: map[int64]*app.PinForecast{3: {
			IsActive:       true,
			LastCycleStart: optional.New(start),
			LastRunTime:    optional.New(start),
		}}}
		info := a.makeInfo(a.colony, f, a.ownerName, a.extraTypeNames, now)
		assert.Equal(t, start.Format(app.DateTimeFormat), value(t, info.general, "Last activity"))
		assert.False(t, hasField(info.specific, "Idle for"))
	})
	t.Run("should show last production of idle processor", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 3)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		start := now.Add(-3 * time.Hour) // cycle time is 1 hour
		f := &app.ColonyForecast{Pins: map[int64]*app.PinForecast{3: {
			LastCycleStart: optional.New(start),
			LastRunTime:    optional.New(now.Add(-10 * time.Minute)), // last check for inputs
		}}}
		info := a.makeInfo(a.colony, f, a.ownerName, a.extraTypeNames, now)
		assert.Equal(t, start.Add(time.Hour).Format(app.DateTimeFormat), value(t, info.general, "Last activity"))
		assert.Equal(t, ihumanize.Duration(2*time.Hour), value(t, info.specific, "Idle for"))
	})
	t.Run("should show unknown activity of processor that never ran", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 3)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		f := &app.ColonyForecast{Pins: map[int64]*app.PinForecast{3: {
			LastRunTime: optional.New(now.Add(-10 * time.Minute)),
		}}}
		info := a.makeInfo(a.colony, f, a.ownerName, a.extraTypeNames, now)
		assert.Equal(t, "-", value(t, info.general, "Last activity"))
		assert.False(t, hasField(info.specific, "Idle for"))
	})
	t.Run("should show name of unreferenced input from database", func(t *testing.T) {
		info := makeInfo(t, 4)
		require.True(t, info.found)
		assert.Equal(t, "Carbon Compounds 0 / 3,000 (not routed)", value(t, info.specific, "Inputs"))
		assert.Equal(t, "Biofuels x 20", value(t, info.specific, "Schematic"))
	})
	t.Run("should show fallback for input missing in database", func(t *testing.T) {
		info := makeInfo(t, 5)
		require.True(t, info.found)
		assert.Equal(t, "Type #2305 0 / 3,000 (not routed)", value(t, info.specific, "Inputs"))
		assert.Equal(t, int64(2305), field(t, info.specific, "Inputs").lines[0].typeID, "unknown types still link")
	})
	t.Run("should show storage", func(t *testing.T) {
		info := makeInfo(t, 2)
		require.True(t, info.found)
		assert.Equal(t, "-", value(t, info.general, "Status"))
		assert.Contains(t, value(t, info.specific, "Capacity"), " / 12,000 m3")
		contents := field(t, info.specific, "Contents")
		require.Len(t, contents.lines, 1)
		assert.Equal(t, "Aqueous Liquids", contents.lines[0].name)
		assert.Equal(t, aqueousLiquids.ID, contents.lines[0].typeID)
		assert.Contains(t, contents.lines[0].detail, " m3)")
		assert.Equal(t, "Aqueous Liquids x 3,000 to Basic Processor (Water)", value(t, info.routes, "Outgoing routes"))
	})
	t.Run("should report missing installation", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 99)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		require.Len(t, a.content.Objects, 1)
		assert.Equal(t, "Installation no longer exists", a.content.Objects[0].(*widget.Label).Text)
	})
	t.Run("should recalculate forecast", func(t *testing.T) {
		a := newColonyPinDetails(u, character.ID, cp.EvePlanet.ID, 1)
		t.Cleanup(a.stop)
		require.NoError(t, a.Update(t.Context()))
		a.content.Objects = nil
		a.refreshForecast()
		assert.NotEmpty(t, a.content.Objects)
	})
}
