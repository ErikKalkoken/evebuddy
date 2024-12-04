package app_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

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
			assert.Equal(t, tc.want, x)
		})
	}
}

func TestEvePlanetTypeDisplay2(t *testing.T) {
	ep := app.EvePlanet{}
	assert.Equal(t, "", ep.TypeDisplay())
}
