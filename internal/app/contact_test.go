package app_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestStandingCategoryString(t *testing.T) {
	cases := []struct {
		name string
		sc   app.StandingCategory
		want string
	}{
		{"terrible", app.TerribleStanding, "terrible"},
		{"bad", app.BadStanding, "bad"},
		{"neutral", app.NeutralStanding, "neutral"},
		{"good", app.GoodStanding, "good"},
		{"excellent", app.ExcellentStanding, "excellent"},
		{"unknown", app.StandingCategory(99), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.sc.String())
		})
	}
}

func TestNewStandingCategory(t *testing.T) {
	cases := []struct {
		v    float64
		want app.StandingCategory
	}{
		{-10, app.TerribleStanding},
		{-0.1, app.BadStanding},
		{0, app.NeutralStanding},
		{5, app.GoodStanding},
		{10, app.ExcellentStanding},
	}
	for _, tc := range cases {
		t.Run(tc.want.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, app.NewStandingCategory(tc.v))
		})
	}
	t.Run("should panic for NaN", func(t *testing.T) {
		assert.Panics(t, func() {
			app.NewStandingCategory(math.NaN())
		})
	})
}
