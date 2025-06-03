package app_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestEveAlliance(t *testing.T) {
	x := &app.EveAlliance{
		ID:   42,
		Name: "name",
	}
	ee := x.ToEveEntity()
	assert.EqualValues(t, 42, ee.ID)
	assert.EqualValues(t, "name", ee.Name)
	assert.EqualValues(t, app.EveEntityAlliance, ee.Category)
}

func TestEveCharacterAlliance(t *testing.T) {
	x1 := &app.EveCharacter{
		Alliance: &app.EveEntity{
			Name: "alliance",
		},
	}
	assert.Equal(t, "alliance", x1.AllianceName())
	assert.True(t, x1.HasAlliance())

	x2 := &app.EveCharacter{}
	assert.Equal(t, "", x2.AllianceName())
	assert.False(t, x2.HasAlliance())
}

func TestEveCharacterDescription(t *testing.T) {
	x := &app.EveCharacter{Description: "alpha<br>bravo"}
	assert.Equal(t, "alpha\nbravo", x.DescriptionPlain())
}

func TestEveCharacterFaction(t *testing.T) {
	x1 := &app.EveCharacter{
		Faction: &app.EveEntity{
			Name: "faction",
		},
	}
	assert.Equal(t, "faction", x1.FactionName())
	assert.True(t, x1.HasFaction())

	x2 := &app.EveCharacter{}
	assert.Equal(t, "", x2.FactionName())
	assert.False(t, x2.HasFaction())
}

func TestEveCharacterRace(t *testing.T) {
	x1 := &app.EveCharacter{Race: &app.EveRace{Description: "description"}}
	assert.Equal(t, "description", x1.RaceDescription())

	x2 := &app.EveCharacter{}
	assert.Equal(t, "", x2.RaceDescription())
}

func TestEveCharacterEveEntity(t *testing.T) {
	x1 := &app.EveCharacter{ID: 42, Name: "name"}
	x2 := x1.ToEveEntity()
	assert.EqualValues(t, 42, x2.ID)
	assert.EqualValues(t, "name", x2.Name)
	assert.EqualValues(t, app.EveEntityCharacter, x2.Category)
}

func TestEveConstellationEveEntity(t *testing.T) {
	x1 := &app.EveConstellation{ID: 42, Name: "name"}
	x2 := x1.ToEveEntity()
	assert.EqualValues(t, 42, x2.ID)
	assert.EqualValues(t, "name", x2.Name)
	assert.EqualValues(t, app.EveEntityConstellation, x2.Category)
}

func TestEveCorporationAlliance(t *testing.T) {
	x1 := &app.EveCorporation{Alliance: &app.EveEntity{}}
	assert.True(t, x1.HasAlliance())

	x2 := &app.EveCorporation{}
	assert.False(t, x2.HasAlliance())
}

func TestEveCorporationFaction(t *testing.T) {
	x1 := &app.EveCorporation{Faction: &app.EveEntity{}}
	assert.True(t, x1.HasFaction())

	x2 := &app.EveCorporation{}
	assert.False(t, x2.HasFaction())
}

func TestEveCorporationEveEntity(t *testing.T) {
	x1 := &app.EveCorporation{ID: 42, Name: "name"}
	x2 := x1.ToEveEntity()
	assert.EqualValues(t, 42, x2.ID)
	assert.EqualValues(t, "name", x2.Name)
	assert.EqualValues(t, app.EveEntityCorporation, x2.Category)
}

func TestEveCorporationDescription(t *testing.T) {
	x := &app.EveCorporation{Description: "alpha<br>bravo"}
	assert.Equal(t, "alpha\nbravo", x.DescriptionPlain())
}

func TestEveEntityCategory(t *testing.T) {
	x := &app.EveEntity{Category: app.EveEntityAlliance}
	assert.Equal(t, "Alliance", x.CategoryDisplay())
}

func TestEveEntityIsCharacter(t *testing.T) {
	x1 := &app.EveEntity{Category: app.EveEntityCharacter}
	assert.True(t, x1.IsCharacter())
	x2 := &app.EveEntity{Category: app.EveEntityAlliance}
	assert.False(t, x2.IsCharacter())
}

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
			assert.Equal(t, tc.want, tc.in.ToEveEntity())
		})
	}
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

func TestEveEntityIsNPC(t *testing.T) {
	cases := []struct {
		name     string
		id       int32
		category app.EveEntityCategory
		want     optional.Optional[bool]
	}{
		{"npc character", 3_000_001, app.EveEntityCharacter, optional.From(true)},
		{"non-npc character", 10_000_001, app.EveEntityCharacter, optional.From(false)},
		{"npc corporation", 1_000_001, app.EveEntityCorporation, optional.From(true)},
		{"non-npc character", 5_000_001, app.EveEntityCorporation, optional.From(false)},
		{"some alliance", 5_000_001, app.EveEntityAlliance, optional.Optional[bool]{}},
		{"some type", 5_000_001, app.EveEntityInventoryType, optional.Optional[bool]{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ee := &app.EveEntity{ID: tc.id, Category: tc.category}
			got := ee.IsNPC()
			assert.Equal(t, tc.want, got)
		})
	}
}
