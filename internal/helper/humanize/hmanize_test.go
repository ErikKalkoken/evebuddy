package humanize_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
)

func TestHumanize(t *testing.T) {
	var cases = []struct {
		value    float64
		decimals int
		want     string
	}{
		{99, 2, "99.00"},
		{42.1234, 2, "42.12"},
		{1000, 2, "1.00K"},
		{1234.56, 2, "1.23K"},
		{1234.56, 0, "1K"},
		{1234000.56, 2, "1.23M"},
		{1234000000.56, 2, "1.23B"},
		{1234000000000.56, 2, "1.23T"},
		{-1234000.56, 2, "-1.23M"},
		{0, 2, "0.00"},
		{1234.56, 3, "1.235K"},
		{1234.56, 1, "1.2K"},
		{1234.56, 0, "1K"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("Can format numbers: %f", tc.value), func(t *testing.T) {
			got := humanize.Number(tc.value, tc.decimals)
			assert.Equal(t, tc.want, got)
		})
	}
}
