package app_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestEveLocationVariantFromID(t *testing.T) {
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

func TestEveLocationDisplayName(t *testing.T) {
	cases := []struct {
		description string
		id          int64
		name        string
		ess         *app.EveSolarSystem
		want        string
	}{
		{"unknown", 888, "", nil, "Unknown"},
		{"asset safety", 2004, "", nil, "Asset Safety"},
		{"known solar system", 30000001, "", &app.EveSolarSystem{Name: "system"}, "system"},
		{"unknown solar system", 30000001, "", nil, "Unknown solar system 30000001"},
		{"station", 60000001, "name", nil, "name"},
		{"unknown structure", 1000000000001, "", nil, "Unknown structure 1000000000001"},
		{"unknown location", 60000001, "", nil, "Unknown location 60000001"},
	}
	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			x := app.EveLocation{
				ID:          tc.id,
				Name:        tc.name,
				SolarSystem: tc.ess,
			}
			assert.Equal(t, tc.want, x.DisplayName())
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

func TestEveLocationRegionName(t *testing.T) {
	o1 := app.EveLocation{
		SolarSystem: &app.EveSolarSystem{
			Constellation: &app.EveConstellation{
				Region: &app.EveRegion{
					Name: "region",
				},
			},
		},
	}
	assert.Equal(t, "region", o1.RegionName())
	assert.Equal(t, "", app.EveLocation{}.RegionName())
}

func TestEveLocationSolarSystemName(t *testing.T) {
	o1 := app.EveLocation{
		SolarSystem: &app.EveSolarSystem{
			Name: "system",
		},
	}
	assert.Equal(t, "system", o1.SolarSystemName())
	assert.Equal(t, "", app.EveLocation{}.SolarSystemName())
}

func TestEveLocationToEveEntity(t *testing.T) {
	cases := []struct {
		name string
		in   *app.EveLocation
		want *app.EveEntity
	}{
		{
			"solar system",
			&app.EveLocation{
				ID: 30000001,
				SolarSystem: &app.EveSolarSystem{
					Name: "system",
				},
			},
			&app.EveEntity{
				ID:       30000001,
				Name:     "system",
				Category: app.EveEntitySolarSystem,
			},
		},
		{
			"station",
			&app.EveLocation{
				ID:   60000001,
				Name: "station",
			},
			&app.EveEntity{
				ID:       60000001,
				Name:     "station",
				Category: app.EveEntityStation,
			},
		},
		{
			"structure",
			&app.EveLocation{
				ID:   1000000000001,
				Name: "structure",
			},
			nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.in.EveEntity())
		})
	}
}
