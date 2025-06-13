package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

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
