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
		out model.LocationVariant
	}{
		{5, model.LocationVariantUnknown},
		{2004, model.LocationVariantAssetSafety},
		{30000142, model.LocationVariantSolarSystem},
		{60003760, model.LocationVariantStation},
		{1042043617604, model.LocationVariantStructure},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("id: %d", tc.in), func(t *testing.T) {
			assert.Equal(t, tc.out, model.LocationVariantFromID(tc.in))
		})
	}
}
