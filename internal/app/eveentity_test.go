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

func TestEveEntity_ToEveEntity(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var ee *app.EveEntity
		assert.Nil(t, ee.ToEveEntity())
	})
	t.Run("returns a clone", func(t *testing.T) {
		ee := &app.EveEntity{ID: 42, Name: "Alpha", Category: app.EveEntityCharacter}
		got := ee.ToEveEntity()
		xassert.Equal(t, ee, got)
		assert.NotSame(t, ee, got)
	})
}

func TestEveEntity_IDOrZero(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var ee *app.EveEntity
		xassert.Equal(t, 0, ee.IDOrZero())
	})
	t.Run("not nil", func(t *testing.T) {
		ee := &app.EveEntity{ID: 42}
		xassert.Equal(t, 42, ee.IDOrZero())
	})
}

func TestEveEntity_NameOrZero(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var ee *app.EveEntity
		xassert.Equal(t, "", ee.NameOrZero())
	})
	t.Run("not nil", func(t *testing.T) {
		ee := &app.EveEntity{Name: "Alpha"}
		xassert.Equal(t, "Alpha", ee.NameOrZero())
	})
}

func TestEveEntity_Category(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var x *app.EveEntity
		xassert.Equal(t, "?", x.CategoryDisplay())
	})
	t.Run("not nil", func(t *testing.T) {
		x := &app.EveEntity{Category: app.EveEntityAlliance}
		xassert.Equal(t, "Alliance", x.CategoryDisplay())
	})
}

func TestEveEntity_IsValid(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var x *app.EveEntity
		assert.False(t, x.IsValid())
	})
	t.Run("known category", func(t *testing.T) {
		x := &app.EveEntity{Category: app.EveEntityCharacter}
		assert.True(t, x.IsValid())
	})
	t.Run("unknown category", func(t *testing.T) {
		x := &app.EveEntity{Category: app.EveEntityUnknown}
		assert.False(t, x.IsValid())
	})
}

func TestEveEntity_IsCharacter(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var x *app.EveEntity
		assert.False(t, x.IsCharacter())
	})
	x1 := &app.EveEntity{Category: app.EveEntityCharacter}
	assert.True(t, x1.IsCharacter())
	x2 := &app.EveEntity{Category: app.EveEntityAlliance}
	assert.False(t, x2.IsCharacter())
}

func TestEveEntityCategory_IsKnown(t *testing.T) {
	assert.True(t, app.EveEntityCharacter.IsKnown())
	assert.False(t, app.EveEntityUndefined.IsKnown())
	assert.False(t, app.EveEntityUnknown.IsKnown())
}

func TestEveEntityCategory_String(t *testing.T) {
	xassert.Equal(t, "character", app.EveEntityCharacter.String())
	xassert.Equal(t, "?", app.EveEntityCategory(99).String())
}

func TestEveEntity_Compare(t *testing.T) {
	t.Run("should order by name", func(t *testing.T) {
		x1 := &app.EveEntity{Name: "Alpha"}
		x2 := &app.EveEntity{Name: "Bravo"}
		assert.Negative(t, x1.Compare(x2))
		assert.Positive(t, x2.Compare(x1))
		assert.Zero(t, x1.Compare(x1))
	})
	t.Run("should not panic when other is nil", func(t *testing.T) {
		x1 := &app.EveEntity{Name: "Alpha"}
		assert.Zero(t, x1.Compare(nil))
	})
	t.Run("should not panic when receiver is nil", func(t *testing.T) {
		var x1 *app.EveEntity
		x2 := &app.EveEntity{Name: "Bravo"}
		assert.Zero(t, x1.Compare(x2))
	})
	t.Run("should not panic when both are nil", func(t *testing.T) {
		var x1 *app.EveEntity
		assert.Zero(t, x1.Compare(nil))
	})
}

func TestEveEntity_IsNPC(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var x *app.EveEntity
		assert.True(t, x.IsNPC().IsEmpty())
	})
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
	t.Run("nil receiver", func(t *testing.T) {
		var ee *app.EveEntity
		_, err := ee.InfoLink()
		assert.Error(t, err)
	})
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
