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
