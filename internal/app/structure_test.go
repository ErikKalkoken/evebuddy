package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

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
