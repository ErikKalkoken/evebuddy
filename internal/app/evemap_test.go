package app_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveConstellationEveEntity(t *testing.T) {
	x1 := &app.EveConstellation{ID: 42, Name: "name"}
	x2 := x1.ToEveEntity()
	xassert.Equal(t, 42, x2.ID)
	xassert.Equal(t, "name", x2.Name)
	xassert.Equal(t, app.EveEntityConstellation, x2.Category)
}

func TestEveRegionDescriptionPlain(t *testing.T) {
	x := app.EveRegion{Description: optional.New("alpha<br>bravo")}
	xassert.Equal(t, "alpha\nbravo", x.DescriptionPlain())
}

func TestEveRegionToEveEntity(t *testing.T) {
	x1 := app.EveRegion{ID: 42, Name: "name"}
	x2 := x1.ToEveEntity()
	xassert.Equal(t, 42, x2.ID)
	xassert.Equal(t, "name", x2.Name)
	xassert.Equal(t, app.EveEntityRegion, x2.Category)
}

func TestSolarSystemSecurityTypeToImportance(t *testing.T) {
	cases := []struct {
		t    app.SolarSystemSecurityType
		want widget.Importance
	}{
		{app.SuperHighSec, widget.SuccessImportance},
		{app.HighSec, widget.SuccessImportance},
		{app.LowSec, widget.WarningImportance},
		{app.NullSec, widget.DangerImportance},
		{app.SolarSystemSecurityType(99), widget.MediumImportance},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.t), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.t.ToImportance())
		})
	}
}

func TestSolarSystemSecurityTypeToColorName(t *testing.T) {
	cases := []struct {
		t    app.SolarSystemSecurityType
		want fyne.ThemeColorName
	}{
		{app.SuperHighSec, theme.ColorNameSuccess},
		{app.HighSec, theme.ColorNameSuccess},
		{app.LowSec, theme.ColorNameWarning},
		{app.NullSec, theme.ColorNameError},
		{app.SolarSystemSecurityType(99), theme.ColorNameForeground},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.t), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.t.ToColorName())
		})
	}
}

func TestNewSolarSystemSecurityTypeFromValue(t *testing.T) {
	cases := []struct {
		v    float32
		want app.SolarSystemSecurityType
	}{
		{0.95, app.SuperHighSec},
		{0.5, app.HighSec},
		{0.1, app.LowSec},
		{0, app.NullSec},
		{-0.5, app.NullSec},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.v), func(t *testing.T) {
			xassert.Equal(t, tc.want, app.NewSolarSystemSecurityTypeFromValue(tc.v))
		})
	}
}

func TestEveSolarSystemIsWormholeSpace(t *testing.T) {
	xassert.Equal(t, true, app.EveSolarSystem{ID: 31000005}.IsWormholeSpace())
	xassert.Equal(t, false, app.EveSolarSystem{ID: 30000005}.IsWormholeSpace())
}

func TestEveSolarSystemToEveEntity(t *testing.T) {
	x1 := app.EveSolarSystem{ID: 42, Name: "name"}
	x2 := x1.ToEveEntity()
	xassert.Equal(t, 42, x2.ID)
	xassert.Equal(t, "name", x2.Name)
	xassert.Equal(t, app.EveEntitySolarSystem, x2.Category)
}

func TestEveSolarSystemDisplayRichText(t *testing.T) {
	es := app.EveSolarSystem{Name: "system", SecurityStatus: 0.5}
	got := es.DisplayRichText()
	want := []widget.RichTextSegment{
		&widget.TextSegment{
			Text: "0.5",
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameSuccess,
				Inline:    true,
			},
		},
		&widget.TextSegment{Text: "  system"},
	}
	xassert.Equal(t, want, got)
}

func TestEveSolarSystemDisplayRichTextWithRegion(t *testing.T) {
	es := app.EveSolarSystem{
		Name:           "system",
		SecurityStatus: 0.5,
		Constellation: &app.EveConstellation{
			Region: &app.EveRegion{Name: "region"},
		},
	}
	got := es.DisplayRichTextWithRegion()
	want := []widget.RichTextSegment{
		&widget.TextSegment{
			Text: "0.5",
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameSuccess,
				Inline:    true,
			},
		},
		&widget.TextSegment{Text: "  system (region)"},
	}
	xassert.Equal(t, want, got)
}

func TestEveRoutePreferenceString(t *testing.T) {
	xassert.Equal(t, "shortest", app.RouteShorter.String())
	xassert.Equal(t, "secure", app.RouteSafer.String())
	xassert.Equal(t, "insecure", app.RouteLessSecure.String())
	xassert.Equal(t, "", app.EveRoutePreference(99).String())
}

func TestEveRoutePreferenceFromString(t *testing.T) {
	xassert.Equal(t, app.RouteShorter, app.EveRoutePreferenceFromString("shortest"))
	xassert.Equal(t, app.RouteSafer, app.EveRoutePreferenceFromString("secure"))
	xassert.Equal(t, app.RouteLessSecure, app.EveRoutePreferenceFromString("insecure"))
	xassert.Equal(t, app.RouteShorter, app.EveRoutePreferenceFromString("unknown"))
}

func TestEveRoutePreferences(t *testing.T) {
	got := app.EveRoutePreferences()
	want := []app.EveRoutePreference{app.RouteShorter, app.RouteSafer, app.RouteLessSecure}
	xassert.Equal(t, want, got)
}

func TestEvePlanetTypeDisplay(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Planet (Gas)", "Gas"},
		{"XXX", ""},
		{"", ""},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprint(i+1), func(t *testing.T) {
			typ := app.EveType{Name: tc.in}
			ep := app.EvePlanet{Type: &typ}
			x := ep.TypeDisplay()
			xassert.Equal(t, tc.want, x)
		})
	}
}

func TestEvePlanetTypeDisplay2(t *testing.T) {
	ep := app.EvePlanet{}
	xassert.Equal(t, "", ep.TypeDisplay())
}
