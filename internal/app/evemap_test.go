package app_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestEveConstellationEveEntity(t *testing.T) {
	x1 := &app.EveConstellation{ID: 42, Name: "name"}
	x2 := x1.EveEntity()
	assert.EqualValues(t, 42, x2.ID)
	assert.EqualValues(t, "name", x2.Name)
	assert.EqualValues(t, app.EveEntityConstellation, x2.Category)
}

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
