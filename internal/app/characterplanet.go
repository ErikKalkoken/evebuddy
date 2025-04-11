package app

import (
	"maps"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type CharacterPlanet struct {
	ID           int64
	CharacterID  int32
	EvePlanet    *EvePlanet
	LastUpdate   time.Time
	LastNotified optional.Optional[time.Time] // expiry time that was last notified
	Pins         []*PlanetPin
	UpgradeLevel int
}

func (cp CharacterPlanet) NameRichText() []widget.RichTextSegment {
	return slices.Concat(
		cp.EvePlanet.SolarSystem.SecurityStatusRichText(),
		iwidget.NewRichTextSegmentFromText("  "+cp.EvePlanet.Name),
	)
}

// ExtractedTypes returns a list of unique types currently being extracted.
func (cp CharacterPlanet) ExtractedTypes() []*EveType {
	types := make(map[int32]*EveType)
	for _, p := range cp.Pins {
		if p.Type.Group.ID != EveGroupExtractorControlUnits || p.ExtractorProductType == nil {
			continue
		}
		types[p.ExtractorProductType.ID] = p.ExtractorProductType
	}
	return slices.Collect(maps.Values(types))
}

func (cp CharacterPlanet) ExtractedTypeNames() []string {
	return extractedStringsSorted(cp.ExtractedTypes(), func(a *EveType) string {
		return a.Name
	})
}

func (cp CharacterPlanet) Extracting() string {
	extractions := strings.Join(cp.ExtractedTypeNames(), ", ")
	if extractions == "" {
		extractions = "-"
	}
	return extractions
}

// ExtractionsExpiryTime returns the final expiry time for all extractions.
// When no expiry data is found it will return a zero time.
func (cp CharacterPlanet) ExtractionsExpiryTime() time.Time {
	expireTimes := make([]time.Time, 0)
	for _, p := range cp.Pins {
		if p.Type.Group.ID != EveGroupExtractorControlUnits {
			continue
		}
		if p.ExpiryTime.IsEmpty() {
			continue
		}
		expireTimes = append(expireTimes, p.ExpiryTime.ValueOrZero())
	}
	if len(expireTimes) == 0 {
		return time.Time{}
	}
	slices.SortFunc(expireTimes, func(a, b time.Time) int {
		return b.Compare(a) // sort descending
	})
	return expireTimes[0]
}

// ProducedSchematics returns a list of unique schematics currently in production.
func (cp CharacterPlanet) ProducedSchematics() []*EveSchematic {
	schematics := make(map[int32]*EveSchematic)
	for _, p := range cp.Pins {
		if p.Type.Group.ID != EveGroupProcessors || p.Schematic == nil {
			continue
		}
		schematics[p.Schematic.ID] = p.Schematic
	}
	return slices.Collect(maps.Values(schematics))
}

func (cp CharacterPlanet) ProducedSchematicNames() []string {
	return extractedStringsSorted(cp.ProducedSchematics(), func(a *EveSchematic) string {
		return a.Name
	})
}

func (cp CharacterPlanet) IsExpired() bool {
	due := cp.ExtractionsExpiryTime()
	if due.IsZero() {
		return false
	}
	return due.Before(time.Now())
}

func (cp CharacterPlanet) Producing() string {
	productions := strings.Join(cp.ProducedSchematicNames(), ", ")
	if productions == "" {
		productions = "-"
	}
	return productions
}

func (cp CharacterPlanet) DueRichText() []widget.RichTextSegment {
	if cp.IsExpired() {
		return iwidget.NewRichTextSegmentFromText("OFFLINE", widget.RichTextStyle{ColorName: theme.ColorNameError})
	}
	due := cp.ExtractionsExpiryTime()
	if due.IsZero() {
		return iwidget.NewRichTextSegmentFromText("-")
	}
	return iwidget.NewRichTextSegmentFromText(due.Format(DateTimeFormat))
}

type PlanetPin struct {
	ID                   int64
	ExpiryTime           optional.Optional[time.Time]
	ExtractorProductType *EveType
	FactorySchematic     *EveSchematic
	InstallTime          optional.Optional[time.Time]
	LastCycleStart       optional.Optional[time.Time]
	Schematic            *EveSchematic
	Type                 *EveType
}

func extractedStringsSorted[T any](s []T, extract func(a T) string) []string {
	s2 := make([]string, 0)
	for _, x := range s {
		s2 = append(s2, extract(x))
	}
	slices.Sort(s2)
	return s2
}
