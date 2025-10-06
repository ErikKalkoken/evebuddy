package app

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

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
		StructureStateAnchorVulnerable:    "anchor vulnerable",
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

func (ss StructureState) IsReinforce() bool {
	return ss == StructureStateArmorReinforce || ss == StructureStateHullReinforce
}

func (ss StructureState) Display() string {
	return xstrings.Title(ss.String())
}

func (ss StructureState) DisplayShort() string {
	var s string
	if ss.IsReinforce() {
		s = "reinforced"
	} else {
		s = ss.String()
	}
	return xstrings.Title(s)

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
	ID                 int64
	Name               string
	NextReinforceApply optional.Optional[time.Time]
	NextReinforceHour  optional.Optional[int64]
	ProfileID          int64
	ReinforceHour      optional.Optional[int64]
	Services           []*StructureService
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

type StructureServiceState uint

const (
	StructureServiceStateUndefined StructureServiceState = iota
	StructureServiceStateOnline
	StructureServiceStateOffline
	StructureServiceStateCleanup
)

type StructureService struct {
	CorporationStructureID int64
	Name                   string
	State                  StructureServiceState
}
