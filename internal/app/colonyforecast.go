package app

import (
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

// ColonyForecastHorizon is how far ahead a forecast looks for when a colony stops working.
const ColonyForecastHorizon = 30 * 24 * time.Hour

// ColonyForecast is the estimated state of a PI colony at a point in time,
// simulated forward from the last ESI snapshot.
type ColonyForecast struct {
	Pins               map[int64]*PinForecast // by pin ID
	Status             ColonyStatus
	Time               time.Time                    // time of the forecast
	WorkEndsAt         optional.Optional[time.Time] // when the colony stops working, if within the horizon
	WorksBeyondHorizon bool                         // colony is still working at the horizon
}

// ProblemStatuses returns the distinct problem statuses of all pins, ordered by status.
func (f ColonyForecast) ProblemStatuses() []PinStatus {
	var s []PinStatus
	for _, p := range f.Pins {
		if p.Status.IsProblem() && !slices.Contains(s, p.Status) {
			s = append(s, p.Status)
		}
	}
	slices.Sort(s)
	return s
}

// PinForecast is the estimated state of a planet pin.
type PinForecast struct {
	Capacity       optional.Optional[float64] // m3, only for storage pins
	CapacityUsed   float64                    // m3
	Contents       map[int64]int64            // amount by type ID
	Demands        map[int64]int64            // input quantity per cycle by type ID, only for factories
	IsActive       bool
	LastCycleStart optional.Optional[time.Time] // start of last production cycle, only for factories
	LastRunTime    optional.Optional[time.Time] // for idle factories the last input check, not the last production
	OutputQuantity int64                        // per cycle, only for factories
	OutputTypeID   int64                        // only for factories
	Status         PinStatus
}

// ColonyStatus is the overall status of a PI colony.
type ColonyStatus uint

const (
	ColonyStatusUndefined ColonyStatus = iota
	ColonyNotSetup
	ColonyNeedsAttention
	ColonyIdle
	ColonyProducing
	ColonyExtracting
)

func (s ColonyStatus) String() string {
	m := map[ColonyStatus]string{
		ColonyStatusUndefined: "undefined",
		ColonyNotSetup:        "not setup",
		ColonyNeedsAttention:  "needs attention",
		ColonyIdle:            "idle",
		ColonyProducing:       "producing",
		ColonyExtracting:      "extracting",
	}
	x, ok := m[s]
	if !ok {
		return "?"
	}
	return x
}

func (s ColonyStatus) Display() string {
	return xstrings.Title(s.String())
}

func (s ColonyStatus) Color() fyne.ThemeColorName {
	switch s {
	case ColonyNotSetup, ColonyNeedsAttention:
		return theme.ColorNameError
	case ColonyIdle:
		return theme.ColorNameWarning
	}
	return theme.ColorNameForeground
}

// IsWorking reports whether the colony is extracting or producing.
func (s ColonyStatus) IsWorking() bool {
	return s == ColonyProducing || s == ColonyExtracting
}

// PinStatus is the status of a planet pin.
type PinStatus uint

const (
	PinStatusUndefined PinStatus = iota
	PinExtracting
	PinExtractorExpired
	PinExtractorInactive
	PinFactoryIdle
	PinInputNotRouted
	PinNotSetup
	PinOutputNotRouted
	PinProducing
	PinStatic
	PinStorageFull
)

func (s PinStatus) String() string {
	m := map[PinStatus]string{
		PinStatusUndefined:   "undefined",
		PinExtracting:        "extracting",
		PinExtractorExpired:  "expired",
		PinExtractorInactive: "inactive",
		PinFactoryIdle:       "idle",
		PinInputNotRouted:    "input not routed",
		PinNotSetup:          "not setup",
		PinOutputNotRouted:   "output not routed",
		PinProducing:         "producing",
		PinStatic:            "static",
		PinStorageFull:       "storage full",
	}
	x, ok := m[s]
	if !ok {
		return "?"
	}
	return x
}

func (s PinStatus) Display() string {
	return xstrings.Title(s.String())
}

func (s PinStatus) Color() fyne.ThemeColorName {
	switch {
	case s.IsProblem():
		return theme.ColorNameError
	case s == PinFactoryIdle:
		return theme.ColorNameWarning
	}
	return theme.ColorNameForeground
}

// IsProblem reports whether the pin needs the player's attention.
func (s PinStatus) IsProblem() bool {
	switch s {
	case
		PinExtractorExpired,
		PinExtractorInactive,
		PinInputNotRouted,
		PinNotSetup,
		PinOutputNotRouted,
		PinStorageFull:
		return true
	}
	return false
}

// IndicatorColor returns the color for indicating the status, e.g. in a symbol.
func (s PinStatus) IndicatorColor() fyne.ThemeColorName {
	switch s {
	case PinExtracting, PinProducing:
		return theme.ColorNameSuccess
	case PinStatic, PinStatusUndefined:
		return theme.ColorNameButton
	}
	return s.Color()
}
