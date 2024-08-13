package app_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestLocationVariantFromID(t *testing.T) {
	cases := []struct {
		in  int64
		out app.EveLocationVariant
	}{
		{5, app.EveLocationUnknown},
		{2004, app.EveLocationAssetSafety},
		{30000142, app.EveLocationSolarSystem},
		{60003760, app.EveLocationStation},
		{1042043617604, app.EveLocationStructure},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("id: %d", tc.in), func(t *testing.T) {
			assert.Equal(t, tc.out, app.LocationVariantFromID(tc.in))
		})
	}
}

func TestEveLocation(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"Alpha - Bravo", "Bravo"},
		{"Alpha - Bravo - Charlie", "Bravo"},
		{"Bravo", "Bravo"},
	}
	for _, tc := range cases {
		t.Run("can return structure name without location", func(t *testing.T) {
			x := app.EveLocation{
				ID:   1_000_000_000_001,
				Name: tc.in,
			}
			assert.Equal(t, tc.out, x.DisplayName2())
		})
	}
}
