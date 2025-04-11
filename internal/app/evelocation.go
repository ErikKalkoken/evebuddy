package app

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type EveLocationVariant int

const (
	EveLocationUnknown EveLocationVariant = iota
	EveLocationAssetSafety
	EveLocationStation
	EveLocationStructure
	EveLocationSolarSystem
)

const LocationUnknownID = 888 // custom ID to signify a location that is not known

// EveLocation is a location in Eve Online.
type EveLocation struct {
	ID          int64
	SolarSystem *EveSolarSystem
	Type        *EveType
	Name        string
	Owner       *EveEntity
	UpdatedAt   time.Time
}

// DisplayName returns a user friendly name.
func (el EveLocation) DisplayName() string {
	if el.Name != "" {
		return el.Name
	}
	return el.alternativeName()
}

func (el EveLocation) DisplayRichText() []widget.RichTextSegment {
	var n string
	if el.Name != "" {
		n = el.Name
	} else {
		n = el.alternativeName()
	}
	if el.SolarSystem == nil {
		return iwidget.NewRichTextSegmentFromText(n)
	}
	return slices.Concat(
		el.SolarSystem.SecurityStatusRichText(),
		iwidget.NewRichTextSegmentFromText(fmt.Sprintf("  %s", n)))
}

// DisplayName2 returns a user friendly name not including the sytem name.
func (el EveLocation) DisplayName2() string {
	if el.Name != "" {
		if el.Variant() != EveLocationStructure {
			return el.Name
		}
		p := strings.Split(el.Name, " - ")
		if len(p) < 2 {
			return el.Name
		}
		return p[1]
	}
	return el.alternativeName()
}

func (el EveLocation) alternativeName() string {
	switch el.Variant() {
	case EveLocationUnknown:
		return "Unknown"
	case EveLocationAssetSafety:
		return "Asset Safety"
	case EveLocationSolarSystem:
		if el.SolarSystem == nil {
			return fmt.Sprintf("Unknown solar system #%d", el.ID)
		}
		return el.SolarSystem.Name
	case EveLocationStructure:
		return fmt.Sprintf("Unknown structure #%d", el.ID)
	}
	return fmt.Sprintf("Unknown location #%d", el.ID)
}

func (el EveLocation) Variant() EveLocationVariant {
	return LocationVariantFromID(el.ID)
}

func LocationVariantFromID(id int64) EveLocationVariant {
	switch {
	case id == LocationUnknownID:
		return EveLocationUnknown
	case id == 2004:
		return EveLocationAssetSafety
	case id >= 30_000_000 && id < 33_000_000:
		return EveLocationSolarSystem
	case id >= 60_000_000 && id < 64_000_000:
		return EveLocationStation
	case id >= 1_000_000_000_000:
		return EveLocationStructure
	default:
		return EveLocationUnknown
	}
}

func (el EveLocation) ToEveEntity() *EveEntity {
	switch el.Variant() {
	case EveLocationSolarSystem:
		return &EveEntity{ID: int32(el.ID), Name: el.Name, Category: EveEntitySolarSystem}
	case EveLocationStation:
		return &EveEntity{ID: int32(el.ID), Name: el.Name, Category: EveEntityStation}
	}
	return nil
}

func (el EveLocation) ToShort() *EveLocationShort {
	o := &EveLocationShort{
		ID:   el.ID,
		Name: optional.New(el.Name),
	}
	if el.SolarSystem != nil {
		o.SecurityStatus = optional.New(el.SolarSystem.SecurityStatus)
	}
	return o
}

// EveLocationShort is a shortended representation of EveLocation.
type EveLocationShort struct {
	ID             int64
	Name           optional.Optional[string]
	SecurityStatus optional.Optional[float32]
}

func (l EveLocationShort) DisplayRichText() []widget.RichTextSegment {
	var s []widget.RichTextSegment
	if !l.SecurityStatus.IsEmpty() {
		secValue := l.SecurityStatus.MustValue()
		secType := NewSolarSystemSecurityTypeFromValue(secValue)
		s = slices.Concat(s, iwidget.NewRichTextSegmentFromText(
			fmt.Sprintf("%.1f", secValue),
			widget.RichTextStyle{ColorName: secType.ToColorName(), Inline: true},
		))
	}
	var name string
	if len(s) > 0 {
		name += "   "
	}
	name += humanize.Optional(l.Name, "?")
	s = slices.Concat(s, iwidget.NewRichTextSegmentFromText(name))
	return s
}

func (l EveLocationShort) SecurityType() optional.Optional[SolarSystemSecurityType] {
	if l.SecurityStatus.IsEmpty() {
		return optional.Optional[SolarSystemSecurityType]{}
	}
	return optional.New(NewSolarSystemSecurityTypeFromValue(l.SecurityStatus.MustValue()))
}
