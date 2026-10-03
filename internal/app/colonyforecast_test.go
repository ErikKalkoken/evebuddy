package app_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestColonyStatusString(t *testing.T) {
	xassert.Equal(t, "needs attention", app.ColonyNeedsAttention.String())
	xassert.Equal(t, "undefined", app.ColonyStatusUndefined.String())
	xassert.Equal(t, "?", app.ColonyStatus(99).String())
}

func TestColonyStatusDisplay(t *testing.T) {
	xassert.Equal(t, "Needs Attention", app.ColonyNeedsAttention.Display())
}

func TestColonyStatusColor(t *testing.T) {
	cases := []struct {
		s    app.ColonyStatus
		want fyne.ThemeColorName
	}{
		{app.ColonyNotSetup, theme.ColorNameError},
		{app.ColonyNeedsAttention, theme.ColorNameError},
		{app.ColonyIdle, theme.ColorNameWarning},
		{app.ColonyProducing, theme.ColorNameForeground},
		{app.ColonyExtracting, theme.ColorNameForeground},
		{app.ColonyStatusUndefined, theme.ColorNameForeground},
	}
	for _, tc := range cases {
		t.Run(tc.s.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.Color())
		})
	}
}

func TestColonyStatusIsWorking(t *testing.T) {
	cases := []struct {
		s    app.ColonyStatus
		want bool
	}{
		{app.ColonyStatusUndefined, false},
		{app.ColonyNotSetup, false},
		{app.ColonyNeedsAttention, false},
		{app.ColonyIdle, false},
		{app.ColonyProducing, true},
		{app.ColonyExtracting, true},
	}
	for _, tc := range cases {
		t.Run(tc.s.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.IsWorking())
		})
	}
}

func TestColonyStatusIsProblem(t *testing.T) {
	cases := []struct {
		s    app.ColonyStatus
		want bool
	}{
		{app.ColonyStatusUndefined, false},
		{app.ColonyNotSetup, true},
		{app.ColonyNeedsAttention, true},
		{app.ColonyIdle, false},
		{app.ColonyProducing, false},
		{app.ColonyExtracting, false},
	}
	for _, tc := range cases {
		t.Run(tc.s.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.IsProblem())
		})
	}
}

func TestPinStatusString(t *testing.T) {
	xassert.Equal(t, "input not routed", app.PinInputNotRouted.String())
	xassert.Equal(t, "undefined", app.PinStatusUndefined.String())
	xassert.Equal(t, "?", app.PinStatus(99).String())
}

func TestPinStatusDisplay(t *testing.T) {
	xassert.Equal(t, "Input Not Routed", app.PinInputNotRouted.Display())
}

func TestPinStatusColor(t *testing.T) {
	cases := []struct {
		s    app.PinStatus
		want fyne.ThemeColorName
	}{
		{app.PinExtractorExpired, theme.ColorNameError},
		{app.PinExtractorInactive, theme.ColorNameError},
		{app.PinInputNotRouted, theme.ColorNameError},
		{app.PinNotSetup, theme.ColorNameError},
		{app.PinOutputNotRouted, theme.ColorNameError},
		{app.PinStorageFull, theme.ColorNameError},
		{app.PinFactoryIdle, theme.ColorNameForeground},
		{app.PinExtracting, theme.ColorNameForeground},
		{app.PinProducing, theme.ColorNameForeground},
		{app.PinStatic, theme.ColorNameForeground},
		{app.PinStatusUndefined, theme.ColorNameForeground},
	}
	for _, tc := range cases {
		t.Run(tc.s.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.Color())
		})
	}
}

func TestPinStatusIndicatorColor(t *testing.T) {
	cases := []struct {
		s    app.PinStatus
		want fyne.ThemeColorName
	}{
		{app.PinExtracting, theme.ColorNameSuccess},
		{app.PinProducing, theme.ColorNameSuccess},
		{app.PinStatic, theme.ColorNameDisabled},
		{app.PinStatusUndefined, theme.ColorNameButton},
		{app.PinFactoryIdle, theme.ColorNameDisabled},
		{app.PinStorageFull, theme.ColorNameError},
	}
	for _, tc := range cases {
		t.Run(tc.s.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.IndicatorColor())
		})
	}
}

func TestPinStatusIsProblem(t *testing.T) {
	cases := []struct {
		s    app.PinStatus
		want bool
	}{
		{app.PinExtractorExpired, true},
		{app.PinExtractorInactive, true},
		{app.PinInputNotRouted, true},
		{app.PinNotSetup, true},
		{app.PinOutputNotRouted, true},
		{app.PinStorageFull, true},
		{app.PinExtracting, false},
		{app.PinFactoryIdle, false},
		{app.PinProducing, false},
		{app.PinStatic, false},
		{app.PinStatusUndefined, false},
	}
	for _, tc := range cases {
		t.Run(tc.s.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.IsProblem())
		})
	}
}

func TestPinForecastExtractorOutput(t *testing.T) {
	pf := app.PinForecast{ExtractorOutputs: []int64{10, 20, 30}}
	t.Run("should return total output", func(t *testing.T) {
		xassert.Equal(t, 60, pf.ExtractorTotalOutput())
		xassert.Equal(t, 0, app.PinForecast{}.ExtractorTotalOutput())
	})
	t.Run("should return output of cycle", func(t *testing.T) {
		xassert.Equal(t, optional.New[int64](20), pf.ExtractorCycleOutput(1))
		assert.True(t, pf.ExtractorCycleOutput(-1).IsEmpty())
		assert.True(t, pf.ExtractorCycleOutput(3).IsEmpty())
	})
}
