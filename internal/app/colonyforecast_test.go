package app_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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
		{app.PinFactoryIdle, theme.ColorNameWarning},
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
