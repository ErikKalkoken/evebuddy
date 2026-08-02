package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveCorporation_DescriptionPlain(t *testing.T) {
	x := &app.EveCorporation{Description: "alpha<br>bravo"}
	xassert.Equal(t, "alpha\nbravo", x.DescriptionPlain())
}

func TestEveEntity_Category(t *testing.T) {
	x := &app.EveEntity{Category: app.EveEntityAlliance}
	xassert.Equal(t, "Alliance", x.CategoryDisplay())
}

func TestEveEntity_IsCharacter(t *testing.T) {
	x1 := &app.EveEntity{Category: app.EveEntityCharacter}
	assert.True(t, x1.IsCharacter())
	x2 := &app.EveEntity{Category: app.EveEntityAlliance}
	assert.False(t, x2.IsCharacter())
}

func TestEveEntity_IsNPC(t *testing.T) {
	cases := []struct {
		name     string
		id       int64
		category app.EveEntityCategory
		want     optional.Optional[bool]
	}{
		{"npc character", 3_000_001, app.EveEntityCharacter, optional.New(true)},
		{"non-npc character", 10_000_001, app.EveEntityCharacter, optional.New(false)},
		{"npc corporation", 1_000_001, app.EveEntityCorporation, optional.New(true)},
		{"non-npc character", 5_000_001, app.EveEntityCorporation, optional.New(false)},
		{"some alliance", 5_000_001, app.EveEntityAlliance, optional.Optional[bool]{}},
		{"some type", 5_000_001, app.EveEntityInventoryType, optional.Optional[bool]{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ee := &app.EveEntity{ID: tc.id, Category: tc.category}
			got := ee.IsNPC()
			xassert.Equal(t, tc.want, got)
		})
	}
}

func TestEveEntity_InfoLink(t *testing.T) {
	cases := []struct {
		category app.EveEntityCategory
		wantLink string
		wantErr  bool
	}{
		{app.EveEntityAlliance, "showinfo:16159//42", false},
		{app.EveEntityCharacter, "showinfo:1373//42", false},
		{app.EveEntityConstellation, "showinfo:4//42", false},
		{app.EveEntityCorporation, "showinfo:2//42", false},
		{app.EveEntityFaction, "showinfo:19//42", false},
		{app.EveEntityInventoryType, "showinfo:42", false},
		{app.EveEntityRegion, "showinfo:3//42", false},
		{app.EveEntitySolarSystem, "showinfo:5//42", false},
		{app.EveEntityStation, "showinfo:54//42", false},
		{app.EveEntityMailList, "", true},
		{app.EveEntityUndefined, "", true},
		{app.EveEntityUnknown, "", true},
	}

	for _, tc := range cases {
		t.Run(tc.category.String(), func(t *testing.T) {
			ee := &app.EveEntity{ID: 42, Category: tc.category}
			gotLink, gotErr := ee.InfoLink()
			if !tc.wantErr {
				assert.Nil(t, gotErr)
				xassert.Equal(t, tc.wantLink, gotLink)
			} else {
				assert.Error(t, gotErr)
			}
		})
	}
}
