package app

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type SolarSystemSecurityType uint

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

const (
	NullSec SolarSystemSecurityType = iota
	LowSec
	HighSec
	SuperHighSec
)

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
	switch v := es.SecurityStatus; {
	case v >= 0.9:
		return SuperHighSec
	case v >= 0.45:
		return HighSec
	case v > 0.0:
		return LowSec
	}
	return NullSec
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
		iwidget.NewRichTextSegmentFromText(fmt.Sprintf("  %s", es.Name), false),
	)
}

type EveSolarSystemPlanet struct {
	AsteroidBeltIDs []int32
	MoonIDs         []int32
	PlanetID        int32
}
