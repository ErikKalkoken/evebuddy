package app

import (
	"cmp"
	"hash/maphash"
	"iter"
	"maps"
	"math"
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
		if v, ok := pp.ProcessorSchematic(); ok {
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

var fingerprintSeed = maphash.MakeSeed()

// Fingerprint returns a hash of the colony data received from ESI.
// Colonies with the same fingerprint have the same forecast. Only valid within the running process.
func (cp CharacterPlanet) Fingerprint() uint64 {
	var h maphash.Hash
	h.SetSeed(fingerprintSeed)
	w := func(v int64) {
		maphash.WriteComparable(&h, v)
	}
	wTime := func(t time.Time) {
		w(t.Unix()) // UnixNano is undefined for the zero time
		w(int64(t.Nanosecond()))
	}
	wBool := func(b bool) {
		if b {
			w(1)
		} else {
			w(0)
		}
	}
	// optional values are written with their presence, so missing and zero differ
	wOptInt := func(o optional.Optional[int64]) {
		v, ok := o.Value()
		wBool(ok)
		if ok {
			w(v)
		}
	}
	wOptTime := func(o optional.Optional[time.Time]) {
		v, ok := o.Value()
		wBool(ok)
		if ok {
			wTime(v)
		}
	}
	typeID := func(et *EveType) int64 {
		if et == nil {
			return 0
		}
		return et.ID
	}
	wOptType := func(o optional.Optional[*EveType]) {
		v, ok := o.Value()
		wBool(ok)
		if ok {
			w(typeID(v))
		}
	}
	wOptSchematic := func(o optional.Optional[*EveSchematic]) {
		v, ok := o.Value()
		wBool(ok && v != nil)
		if ok && v != nil {
			w(v.ID)
		}
	}

	wTime(cp.LastUpdate)
	w(cp.UpgradeLevel)
	pins := sortedIfNeeded(cp.Pins, func(a, b *PlanetPin) int {
		return cmp.Compare(a.ID, b.ID)
	})
	w(int64(len(pins)))
	for _, p := range pins {
		w(p.ID)
		w(typeID(p.Type))
		contents := sortedIfNeeded(p.Contents, func(a, b *PlanetPinContent) int {
			return cmp.Or(cmp.Compare(typeID(a.Type), typeID(b.Type)), cmp.Compare(a.Amount, b.Amount))
		})
		w(int64(len(contents)))
		for _, c := range contents {
			w(typeID(c.Type))
			w(c.Amount)
		}
		wOptTime(p.ExpiryTime)
		cycle, ok := p.ExtractorCycleTime.Value()
		wBool(ok)
		if ok {
			w(int64(cycle))
		}
		radius, ok := p.ExtractorHeadRadius.Value()
		wBool(ok)
		if ok {
			w(int64(math.Float64bits(radius)))
		}
		wOptInt(p.ExtractorNumHeads)
		wOptType(p.ExtractorProductType)
		wOptInt(p.ExtractorQtyPerCycle)
		wOptSchematic(p.FactorySchematic)
		wOptTime(p.InstallTime)
		wOptTime(p.LastCycleStart)
		wOptSchematic(p.Schematic)
	}
	routes := sortedIfNeeded(cp.Routes, func(a, b *PlanetRoute) int {
		return cmp.Compare(a.RouteID, b.RouteID)
	})
	w(int64(len(routes)))
	for _, r := range routes {
		w(r.RouteID)
		w(r.SourcePinID)
		w(r.DestinationPinID)
		w(typeID(r.ContentType))
		w(r.Quantity)
	}
	return h.Sum64()
}

// sortedIfNeeded returns s sorted, without copying when it is already sorted.
func sortedIfNeeded[S ~[]E, E any](s S, cmp func(a, b E) int) S {
	if slices.IsSortedFunc(s, cmp) {
		return s
	}
	s2 := slices.Clone(s)
	slices.SortFunc(s2, cmp)
	return s2
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

// TypeGroupNames returns the group names of all types known to a colony by type ID.
// Types without a group are omitted.
func (cp CharacterPlanet) TypeGroupNames() map[int64]string {
	m := make(map[int64]string)
	for et := range cp.types() {
		if et.Group != nil {
			m[et.ID] = et.Group.Name
		}
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
	_, ok := pp.ProcessorSchematic()
	return pp.Type.Group.ID == EveGroupProcessors && ok
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
