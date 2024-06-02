package model_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestLocationVariantFromID(t *testing.T) {
	cases := []struct {
		in  int64
		out model.EveLocationVariant
	}{
		{5, model.EveLocationUnknown},
		{2004, model.EveLocationAssetSafety},
		{30000142, model.EveLocationSolarSystem},
		{60003760, model.EveLocationStation},
		{1042043617604, model.EveLocationStructure},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("id: %d", tc.in), func(t *testing.T) {
			assert.Equal(t, tc.out, model.LocationVariantFromID(tc.in))
		})
	}
}
