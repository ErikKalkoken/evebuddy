package app

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// type StructureServiceState uint

// const (
// 	StructureServiceStateUndefined StructureServiceState = iota
// 	StructureServiceStateOnline
// 	StructureServiceStateOffline
// 	StructureServiceStateCleanup
// )

type StructureState uint

const (
	StructureStateUndefined StructureState = iota
	StructureStateAnchoring
	StructureStateAnchorVulnerable
	StructureStateArmorReinforce
	StructureStateArmorVulnerable
	StructureStateDeployVulnerable
	StructureStateFittingInvulnerable
	StructureStateHullReinforce
	StructureStateHullVulnerable
	StructureStateOnlineDeprecated
	StructureStateOnliningVulnerable
	StructureStateShieldVulnerable
	StructureStateUnanchored
	StructureStateUnknown
)

func (ss StructureState) String() string {
	m := map[StructureState]string{
		StructureStateUndefined:           "",
		StructureStateAnchoring:           "anchoring",
		StructureStateAnchorVulnerable:    "armor vulnerable",
		StructureStateArmorReinforce:      "armor reinforce",
		StructureStateArmorVulnerable:     "armor vulnerable",
		StructureStateDeployVulnerable:    "deploy vulnerable",
		StructureStateFittingInvulnerable: "fitting invulnerable",
		StructureStateHullReinforce:       "hull reinforce",
		StructureStateHullVulnerable:      "hull vulnerable",
		StructureStateOnlineDeprecated:    "online deprecated",
		StructureStateOnliningVulnerable:  "onlining vulnerable",
		StructureStateShieldVulnerable:    "shield vulnerable",
		StructureStateUnanchored:          "unanchored",
		StructureStateUnknown:             "unknown",
	}
	return m[ss]
}

func (ss StructureState) Display() string {
	titler := cases.Title(language.English)
	return titler.String(ss.String())
}

func (ss StructureState) Color() fyne.ThemeColorName {
	switch ss {
	case StructureStateAnchoring, StructureStateAnchorVulnerable, StructureStateDeployVulnerable:
		return theme.ColorNameWarning
	case StructureStateArmorReinforce, StructureStateHullReinforce:
		return theme.ColorNameError
	case StructureStateShieldVulnerable:
		return theme.ColorNameSuccess
	}
	return theme.ColorNameForeground
}

type CorporationStructure struct {
	CorporationID      int32
	FuelExpires        optional.Optional[time.Time]
	Name               string
	NextReinforceApply optional.Optional[time.Time]
	NextReinforceHour  optional.Optional[int64]
	ProfileID          int64
	ReinforceHour      optional.Optional[int64]
	State              StructureState
	StateTimerEnd      optional.Optional[time.Time]
	StateTimerStart    optional.Optional[time.Time]
	StructureID        int64
	System             *EveSolarSystem
	Type               *EveType
	UnanchorsAt        optional.Optional[time.Time]
}

func (cs CorporationStructure) NameShort() string {
	if cs.System == nil {
		return cs.Name
	}
	return strings.TrimPrefix(cs.Name, fmt.Sprintf("%s -", cs.System.Name))
}
