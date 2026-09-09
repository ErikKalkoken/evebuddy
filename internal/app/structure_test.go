package app_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestStructureStateString(t *testing.T) {
	xassert.Equal(t, "hull reinforce", app.StructureStateHullReinforce.String())
	xassert.Equal(t, "", app.StructureStateUndefined.String())
}

func TestStructureStateIsReinforce(t *testing.T) {
	xassert.Equal(t, true, app.StructureStateArmorReinforce.IsReinforce())
	xassert.Equal(t, true, app.StructureStateHullReinforce.IsReinforce())
	xassert.Equal(t, false, app.StructureStateAnchoring.IsReinforce())
}

func TestStructureStateDisplay(t *testing.T) {
	xassert.Equal(t, "Hull Reinforce", app.StructureStateHullReinforce.Display())
}

func TestStructureStateDisplayShort(t *testing.T) {
	xassert.Equal(t, "Reinforced", app.StructureStateHullReinforce.DisplayShort())
	xassert.Equal(t, "Anchoring", app.StructureStateAnchoring.DisplayShort())
}

func TestStructureStateColor(t *testing.T) {
	cases := []struct {
		s    app.StructureState
		want fyne.ThemeColorName
	}{
		{app.StructureStateAnchoring, theme.ColorNameWarning},
		{app.StructureStateAnchorVulnerable, theme.ColorNameWarning},
		{app.StructureStateDeployVulnerable, theme.ColorNameWarning},
		{app.StructureStateArmorReinforce, theme.ColorNameError},
		{app.StructureStateHullReinforce, theme.ColorNameError},
		{app.StructureStateShieldVulnerable, theme.ColorNameSuccess},
		{app.StructureStateUnknown, theme.ColorNameForeground},
	}
	for _, tc := range cases {
		t.Run(tc.s.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.Color())
		})
	}
}

func TestCorporationStructure_DisplayName(t *testing.T) {
	t.Run("should return name when set", func(t *testing.T) {
		cs := app.CorporationStructure{
			ID:     42,
			Name:   optional.New("1DQ1-A - Keepstar"),
			System: &app.EveSolarSystem{Name: "1DQ1-A"},
		}
		xassert.Equal(t, "1DQ1-A - Keepstar", cs.DisplayName())
	})
	t.Run("should return fallback when name is empty", func(t *testing.T) {
		cs := app.CorporationStructure{
			ID:     42,
			System: &app.EveSolarSystem{Name: "1DQ1-A"},
		}
		xassert.Equal(t, "1DQ1-A - Structure #42", cs.DisplayName())
	})
}

func TestCorporationStructure_NameShort(t *testing.T) {
	t.Run("should strip the system name prefix without a stray space", func(t *testing.T) {
		cs := app.CorporationStructure{
			Name:   optional.New("1DQ1-A - Keepstar"),
			System: &app.EveSolarSystem{Name: "1DQ1-A"},
		}
		xassert.Equal(t, "Keepstar", cs.NameShort())
	})
	t.Run("should return full name when it does not have the system prefix", func(t *testing.T) {
		cs := app.CorporationStructure{
			Name:   optional.New("Keepstar"),
			System: &app.EveSolarSystem{Name: "1DQ1-A"},
		}
		xassert.Equal(t, "Keepstar", cs.NameShort())
	})
	t.Run("should return name as is when system is nil", func(t *testing.T) {
		cs := app.CorporationStructure{
			Name: optional.New("1DQ1-A - Keepstar"),
		}
		xassert.Equal(t, "1DQ1-A - Keepstar", cs.NameShort())
	})
	t.Run("should return empty when name is empty and system is nil", func(t *testing.T) {
		cs := app.CorporationStructure{}
		xassert.Equal(t, "", cs.NameShort())
	})
}
