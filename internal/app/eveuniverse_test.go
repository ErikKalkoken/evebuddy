package app_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

func TestEveLocationDisplayName2(t *testing.T) {
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

func TestEveLocationDisplayRichText(t *testing.T) {
	t.Run("can return as rich text", func(t *testing.T) {
		ss := &app.EveSolarSystem{SecurityStatus: 0.5}
		l := &app.EveLocation{Name: "location_name", SolarSystem: ss}
		got := l.DisplayRichText()
		want := []widget.RichTextSegment{
			&widget.TextSegment{
				Text: "0.5",
				Style: widget.RichTextStyle{
					ColorName: theme.ColorNameSuccess,
					Inline:    true,
				},
			},
			&widget.TextSegment{
				Text: "  location_name",
			},
		}
		assert.Equal(t, want, got)
	})
	t.Run("can handle missing solar system", func(t *testing.T) {
		l := &app.EveLocation{Name: "location_name"}
		got := l.DisplayRichText()
		want := []widget.RichTextSegment{
			&widget.TextSegment{
				Text: "location_name",
			},
		}
		assert.Equal(t, want, got)
	})
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

func TestEveSchematic(t *testing.T) {
	es := &app.EveSchematic{
		ID:   66,
		Name: "Cooliant",
	}
	r, ok := es.Icon()
	if assert.True(t, ok) {
		assert.NotNil(t, r)
	}
}
