package app

import (
	"iter"
	"maps"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// CharacterPlanet is a PI colony of a character.
type CharacterPlanet struct {
	ID           int64
	CharacterID  int64
	EvePlanet    *EvePlanet
	LastUpdate   time.Time
	LastNotified optional.Optional[time.Time] // last update of the snapshot that was last notified
	Pins         []*PlanetPin
	Routes       []*PlanetRoute
	UpgradeLevel int64
}

func (cp CharacterPlanet) NameRichText() []widget.RichTextSegment {
	return slices.Concat(
		cp.EvePlanet.SolarSystem.SecurityStatusRichText(),
		xwidget.RichTextSegmentsFromText("  "+cp.EvePlanet.Name),
	)
}

// ExtractedTypes returns a list of unique types currently being extracted.
func (cp CharacterPlanet) ExtractedTypes() []*EveType {
	types := make(map[int64]*EveType)
	for pp := range cp.ActiveExtractors() {
		if v, ok := pp.ExtractorProductType.Value(); ok {
			types[v.ID] = v
		}
	}
	return slices.Collect(maps.Values(types))
}

func (cp CharacterPlanet) ActiveExtractors() iter.Seq[*PlanetPin] {
	return xiter.Filter(slices.Values(cp.Pins), func(o *PlanetPin) bool {
		return o.IsExtracting()
	})
}

func (cp CharacterPlanet) ActiveProducers() iter.Seq[*PlanetPin] {
	return xiter.Filter(slices.Values(cp.Pins), func(o *PlanetPin) bool {
		return o.IsProducing()
	})
}

// ProducedSchematics returns a list of unique schematics currently in production.
func (cp CharacterPlanet) ProducedSchematics() []*EveSchematic {
	schematics := make(map[int64]*EveSchematic)
	for pp := range cp.ActiveProducers() {
		if v, ok := pp.Schematic.Value(); ok {
			schematics[v.ID] = v
		}
	}
	return slices.Collect(maps.Values(schematics))
}

// Short type names of planet pins.
const (
	PinTypeAdvancedProcessor = "Advanced Processor"
	PinTypeBasicProcessor    = "Basic Processor"
	PinTypeCommandCenter     = "Command Center"
	PinTypeExtractor         = "Extractor"
	PinTypeHighTechProcessor = "High-Tech Processor"
	PinTypeLaunchpad         = "Launchpad"
	PinTypeStorage           = "Storage"
)

var pinShortTypeNames = map[string]string{
	"Advanced Industry Facility": PinTypeAdvancedProcessor,
	"Basic Industry Facility":    PinTypeBasicProcessor,
	"Command Center":             PinTypeCommandCenter,
	"Extractor Control Unit":     PinTypeExtractor,
	"High-Tech Production Plant": PinTypeHighTechProcessor,
	"Launchpad":                  PinTypeLaunchpad,
	"Storage Facility":           PinTypeStorage,
}

// PinTypeName returns the short type name of a pin, e.g. "Extractor".
// Unknown types return their name without the planet type.
func (cp CharacterPlanet) PinTypeName(p *PlanetPin) string {
	n, _ := strings.CutPrefix(p.Type.Name, cp.EvePlanet.TypeDisplay()+" ")
	if s, ok := pinShortTypeNames[n]; ok {
		return s
	}
	return n
}

// PinName returns the name of a pin, e.g. "Extractor H6-3IS".
func (cp CharacterPlanet) PinName(p *PlanetPin) string {
	return cp.PinTypeName(p) + " " + p.Designator()
}

// TypeNames returns the names of all types known to a colony by type ID.
func (cp CharacterPlanet) TypeNames() map[int64]string {
	m := make(map[int64]string)
	for et := range cp.types() {
		m[et.ID] = et.Name
	}
	return m
}

// TypeVolumes returns the volumes of all types known to a colony by type ID.
func (cp CharacterPlanet) TypeVolumes() map[int64]float64 {
	m := make(map[int64]float64)
	for et := range cp.types() {
		m[et.ID] = et.Volume.ValueOrZero()
	}
	return m
}

// types returns all types known to a colony. Types can repeat.
func (cp CharacterPlanet) types() iter.Seq[*EveType] {
	return func(yield func(*EveType) bool) {
		emit := func(et *EveType) bool { return et == nil || yield(et) }
		for _, p := range cp.Pins {
			for _, c := range p.Contents {
				if !emit(c.Type) {
					return
				}
			}
			if v, ok := p.ExtractorProductType.Value(); ok && !emit(v) {
				return
			}
		}
		for _, r := range cp.Routes {
			if !emit(r.ContentType) {
				return
			}
		}
	}
}

type PlanetPin struct {
	ID                   int64
	Contents             []*PlanetPinContent
	ExpiryTime           optional.Optional[time.Time]
	ExtractorCycleTime   optional.Optional[time.Duration]
	ExtractorHeadRadius  optional.Optional[float64]
	ExtractorNumHeads    optional.Optional[int64]
	ExtractorProductType optional.Optional[*EveType]
	ExtractorQtyPerCycle optional.Optional[int64]
	FactorySchematic     optional.Optional[*EveSchematic]
	InstallTime          optional.Optional[time.Time]
	LastCycleStart       optional.Optional[time.Time]
	Schematic            optional.Optional[*EveSchematic]
	Type                 *EveType
}

// Designator returns the designator of a pin as shown in game, e.g. "H6-3IS".
// Same algorithm as RIFT (Nohus, used with permission).
func (pp PlanetPin) Designator() string {
	const characters = "123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const base = len(characters) - 1 // as in game, so Z is never used
	var b strings.Builder
	id := pp.ID
	for i := range 5 {
		b.WriteByte(characters[id%int64(base)])
		id /= int64(base)
		if i == 1 {
			b.WriteByte('-')
		}
	}
	return b.String()
}

func (pp PlanetPin) IsExtracting() bool {
	return pp.Type.Group.ID == EveGroupExtractorControlUnits && !pp.ExtractorProductType.IsEmpty()
}

func (pp PlanetPin) IsProducing() bool {
	return pp.Type.Group.ID == EveGroupProcessors && !pp.Schematic.IsEmpty()
}

// ProcessorSchematic returns the schematic of a processor.
func (pp PlanetPin) ProcessorSchematic() (*EveSchematic, bool) {
	if es, ok := pp.Schematic.Value(); ok {
		return es, true
	}
	return pp.FactorySchematic.Value()
}

// PlanetPinContent is a commodity stored in a planet pin.
type PlanetPinContent struct {
	Amount int64
	Type   *EveType
}

// PlanetRoute is a route for moving commodities between two pins of a colony.
type PlanetRoute struct {
	ContentType      *EveType
	DestinationPinID int64
	Quantity         int64
	RouteID          int64
	SourcePinID      int64
}
