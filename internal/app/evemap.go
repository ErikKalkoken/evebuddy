package app

import (
	"fmt"
	"regexp"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// EveConstellation is a constellation in Eve Online.
type EveConstellation struct {
	ID     int32
	Name   string
	Region *EveRegion
}

func (ec EveConstellation) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityConstellation}
}

// EveRegion is a region in Eve Online.
type EveRegion struct {
	Description string
	ID          int32
	Name        string
}

func (er EveRegion) DescriptionPlain() string {
	return evehtml.ToPlain(er.Description)
}

func (er EveRegion) ToEveEntity() *EveEntity {
	return &EveEntity{ID: er.ID, Name: er.Name, Category: EveEntityRegion}
}

type SolarSystemSecurityType uint

const (
	NullSec SolarSystemSecurityType = iota
	LowSec
	HighSec
	SuperHighSec
)

// ToImportance returns the importance value for a security type.
func (t SolarSystemSecurityType) ToImportance() widget.Importance {
	switch t {
	case SuperHighSec:
		return widget.HighImportance
	case HighSec:
		return widget.SuccessImportance
	case LowSec:
		return widget.WarningImportance
	case NullSec:
		return widget.DangerImportance
	}
	return widget.MediumImportance
}

func (t SolarSystemSecurityType) ToColorName() fyne.ThemeColorName {
	switch t {
	case SuperHighSec:
		return theme.ColorNamePrimary
	case HighSec:
		return theme.ColorNameSuccess
	case LowSec:
		return theme.ColorNameWarning
	case NullSec:
		return theme.ColorNameError
	}
	return theme.ColorNameForeground
}

func NewSolarSystemSecurityTypeFromValue(v float32) SolarSystemSecurityType {
	switch {
	case v >= 0.9:
		return SuperHighSec
	case v >= 0.45:
		return HighSec
	case v > 0.0:
		return LowSec
	}
	return NullSec
}

// EveSolarSystem is a solar system in Eve Online.
type EveSolarSystem struct {
	Constellation  *EveConstellation
	ID             int32
	Name           string
	SecurityStatus float32
}

func (es EveSolarSystem) IsWormholeSpace() bool {
	return es.ID >= 31000000
}

func (es EveSolarSystem) SecurityType() SolarSystemSecurityType {
	return NewSolarSystemSecurityTypeFromValue(es.SecurityStatus)
}

func (es EveSolarSystem) SecurityStatusDisplay() string {
	return fmt.Sprintf("%.1f", es.SecurityStatus)
}

func (es EveSolarSystem) SecurityStatusRichText() []widget.RichTextSegment {
	return []widget.RichTextSegment{&widget.TextSegment{
		Text: es.SecurityStatusDisplay(),
		Style: widget.RichTextStyle{
			ColorName: es.SecurityType().ToColorName(),
			Inline:    true,
		},
	}}
}

func (es EveSolarSystem) ToEveEntity() *EveEntity {
	return &EveEntity{ID: es.ID, Name: es.Name, Category: EveEntitySolarSystem}
}

func (es EveSolarSystem) DisplayRichText() []widget.RichTextSegment {
	return slices.Concat(
		es.SecurityStatusRichText(),
		iwidget.NewRichTextSegmentFromText(fmt.Sprintf("  %s", es.Name)),
	)
}

func (es EveSolarSystem) DisplayRichTextWithRegion() []widget.RichTextSegment {
	return slices.Concat(
		es.SecurityStatusRichText(),
		iwidget.NewRichTextSegmentFromText(fmt.Sprintf("  %s (%s)", es.Name, es.Constellation.Region.Name)),
	)
}

type EveSolarSystemPlanet struct {
	AsteroidBeltIDs []int32
	MoonIDs         []int32
	PlanetID        int32
}

// EveRouteHeader describes the header for a route in EVE Online.
type EveRouteHeader struct {
	Origin      *EveSolarSystem
	Destination *EveSolarSystem
	Preference  EveRoutePreference
}

func (x EveRouteHeader) String() string {
	var originID, destinationID int32
	if x.Origin != nil {
		originID = x.Origin.ID
	}
	if x.Destination != nil {
		destinationID = x.Destination.ID
	}
	return fmt.Sprintf("{Origin: %d Destination: %d Preference: %s}", originID, destinationID, x.Preference)
}

// EveRoutePreference represents the calculation preference when requesting a route from ESI.
type EveRoutePreference uint

const (
	RouteShortest EveRoutePreference = iota
	RouteSecure
	RouteInsecure
)

func (x EveRoutePreference) String() string {
	m := map[EveRoutePreference]string{
		RouteShortest: "shortest",
		RouteSecure:   "secure",
		RouteInsecure: "insecure",
	}
	return m[x]
}

func EveRoutePreferenceFromString(s string) EveRoutePreference {
	m := map[string]EveRoutePreference{
		"shortest": RouteShortest,
		"secure":   RouteSecure,
		"insecure": RouteInsecure,
	}
	return m[s]
}

func EveRoutePreferences() []EveRoutePreference {
	return []EveRoutePreference{RouteShortest, RouteSecure, RouteInsecure}
}

// EveMoon is a moon in Eve Online.
type EveMoon struct {
	ID          int32
	Name        string
	SolarSystem *EveSolarSystem
}

var rePlanetType = regexp.MustCompile(`Planet \((\S*)\)`)

// EvePlanet is a planet in Eve Online.
type EvePlanet struct {
	ID          int32
	Name        string
	SolarSystem *EveSolarSystem
	Type        *EveType
}

func (ep EvePlanet) TypeDisplay() string {
	if ep.Type == nil {
		return ""
	}
	m := rePlanetType.FindStringSubmatch(ep.Type.Name)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}
