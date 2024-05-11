package humanize_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
)

func TestNumber(t *testing.T) {
	var cases = []struct {
		value    float64
		decimals int
		want     string
	}{
		{99, 2, "99.00"},
		{42.1234, 2, "42.12"},
		{1000, 2, "1.00 K"},
		{1234.56, 2, "1.23 K"},
		{1234.56, 0, "1 K"},
		{1234000.56, 2, "1.23 M"},
		{1234000000.56, 2, "1.23 B"},
		{1234000000000.56, 2, "1.23 T"},
		{-1234000.56, 2, "-1.23 M"},
		{0, 2, "0.00"},
		{1234.56, 3, "1.235 K"},
		{1234.56, 1, "1.2 K"},
		{1234.56, 0, "1 K"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("Can format numbers: %f", tc.value), func(t *testing.T) {
			got := humanize.Number(tc.value, tc.decimals)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDuration(t *testing.T) {
	var cases = []struct {
		in  time.Duration
		out string
	}{
		{24*10*time.Hour + 5*time.Hour + 3*time.Minute, "10d 5h"},
		{5*time.Hour + 3*time.Minute, "5h 3m"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("Can format duration: %v", tc.in), func(t *testing.T) {
			got := humanize.Duration(tc.in)
			assert.Equal(t, tc.out, got)
		})
	}
}
