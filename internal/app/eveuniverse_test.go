package app_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveAlliance(t *testing.T) {
	x := &app.EveAlliance{
		ID:   42,
		Name: "name",
	}
	ee := x.ToEveEntity()
	xassert.Equal(t, 42, ee.ID)
	xassert.Equal(t, "name", ee.Name)
	xassert.Equal(t, app.EveEntityAlliance, ee.Category)
}

func TestEveBloodlineLogo(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var eb *app.EveBloodline
		res, ok := eb.Logo()
		assert.False(t, ok)
		assert.Nil(t, res)
	})
	t.Run("unknown bloodline", func(t *testing.T) {
		eb := &app.EveBloodline{ID: -999}
		res, ok := eb.Logo()
		assert.False(t, ok)
		assert.Nil(t, res)
	})
	t.Run("known bloodline", func(t *testing.T) {
		eb := &app.EveBloodline{ID: 1}
		res, ok := eb.Logo()
		assert.True(t, ok)
		assert.NotNil(t, res)
	})
}

func TestEveCharacter_Description(t *testing.T) {
	x := &app.EveCharacter{
		Description: optional.New("alpha<br>bravo"),
	}
	xassert.Equal(t, "alpha\nbravo", x.DescriptionPlain())
}

func TestEveCharacter_CorporationTitlePlain(t *testing.T) {
	x := &app.EveCharacter{
		CorporationTitle: optional.New("alpha<br>bravo"),
	}
	xassert.Equal(t, "alpha\nbravo", x.CorporationTitlePlain())
}

func TestEveCharacter_EntityIDs(t *testing.T) {
	t.Run("without alliance or faction", func(t *testing.T) {
		x := app.EveCharacter{ID: 1, Corporation: &app.EveEntity{ID: 2}}
		got := slices.Collect(x.EntityIDs().All())
		assert.ElementsMatch(t, []int64{1, 2}, got)
	})
	t.Run("with alliance and faction", func(t *testing.T) {
		x := app.EveCharacter{
			ID:          1,
			Corporation: &app.EveEntity{ID: 2},
			Alliance:    optional.New(&app.EveEntity{ID: 3}),
			Faction:     optional.New(&app.EveEntity{ID: 4}),
		}
		got := slices.Collect(x.EntityIDs().All())
		assert.ElementsMatch(t, []int64{1, 2, 3, 4}, got)
	})
}

func TestEveCharacter_EveEntity(t *testing.T) {
	x1 := &app.EveCharacter{ID: 42, Name: "name"}
	x2 := x1.ToEveEntity()
	xassert.Equal(t, 42, x2.ID)
	xassert.Equal(t, "name", x2.Name)
	xassert.Equal(t, app.EveEntityCharacter, x2.Category)
}

func TestEveCharacter_IsIdentical(t *testing.T) {
	t.Run("should report when same", func(t *testing.T) {
		x1 := &app.EveCharacter{
			Alliance:         optional.New(&app.EveEntity{ID: 1}),
			Birthday:         time.Now(),
			Bloodline:        optional.New(&app.EntityShort{ID: 5}),
			Corporation:      &app.EveEntity{ID: 2},
			Description:      optional.New("abc"),
			Faction:          optional.New(&app.EveEntity{ID: 3}),
			Gender:           "male",
			ID:               4,
			Name:             "Bruce Wayne",
			Race:             &app.EveRace{ID: 4},
			SecurityStatus:   optional.New(-4.5),
			CorporationTitle: optional.New("def"),
		}
		x2 := new(app.EveCharacter)
		*x2 = *x1
		assert.True(t, x1.Equal(x2))
	})
	t.Run("should report when not same", func(t *testing.T) {
		x1 := &app.EveCharacter{
			Alliance:         optional.New(&app.EveEntity{ID: 1}),
			Birthday:         time.Now(),
			Corporation:      &app.EveEntity{ID: 2},
			Description:      optional.New("abc"),
			Faction:          optional.New(&app.EveEntity{ID: 3}),
			Gender:           "male",
			ID:               4,
			Name:             "Bruce Wayne",
			Race:             &app.EveRace{ID: 4},
			SecurityStatus:   optional.New(-4.5),
			CorporationTitle: optional.New("def"),
		}
		x2 := new(app.EveCharacter)
		*x2 = *x1
		x2.ID = 3
		assert.False(t, x1.Equal(x2))
	})
	t.Run("should not panic when other is nil", func(t *testing.T) {
		x1 := &app.EveCharacter{ID: 4}
		assert.False(t, x1.Equal(nil))
	})
}

func TestEveCorporation_EveEntity(t *testing.T) {
	x1 := &app.EveCorporation{ID: 42, Name: "name"}
	x2 := x1.ToEveEntity()
	xassert.Equal(t, 42, x2.ID)
	xassert.Equal(t, "name", x2.Name)
	xassert.Equal(t, app.EveEntityCorporation, x2.Category)
}

func TestEveCorporation_IsIdentical(t *testing.T) {
	t.Run("should report when same", func(t *testing.T) {
		x1 := &app.EveCorporation{
			Alliance:    optional.New(&app.EveEntity{ID: 1}),
			Ceo:         optional.New(&app.EveEntity{ID: 2}),
			Creator:     optional.New(&app.EveEntity{ID: 3}),
			DateFounded: optional.New(time.Now().Add(-3 * time.Hour)),
			Description: "abc",
			Faction:     optional.New(&app.EveEntity{ID: 4}),
			HomeStation: optional.New(&app.EveEntity{ID: 5}),
			ID:          6,
			MemberCount: 7,
			Name:        "def",
			Shares:      optional.New[int64](8),
			TaxRate:     9.1,
			Ticker:      "ghi",
			URL:         optional.New("jkl"),
			WarEligible: optional.New(true),
			Timestamp:   time.Now(),
		}
		x2 := x1
		assert.True(t, x1.Equal(x2))
	})
	t.Run("should report when not same", func(t *testing.T) {
		x1 := &app.EveCorporation{
			Alliance:    optional.New(&app.EveEntity{ID: 1}),
			Ceo:         optional.New(&app.EveEntity{ID: 2}),
			Creator:     optional.New(&app.EveEntity{ID: 3}),
			DateFounded: optional.New(time.Now().Add(-3 * time.Hour)),
			Description: "abc",
			Faction:     optional.New(&app.EveEntity{ID: 4}),
			HomeStation: optional.New(&app.EveEntity{ID: 5}),
			ID:          6,
			MemberCount: 7,
			Name:        "def",
			Shares:      optional.New[int64](8),
			TaxRate:     9.1,
			Ticker:      "ghi",
			URL:         optional.New("jkl"),
			WarEligible: optional.New(true),
			Timestamp:   time.Now(),
		}
		x2 := &app.EveCorporation{
			ID: 4,
		}
		assert.False(t, x1.Equal(x2))
	})
	t.Run("should not panic when other is nil", func(t *testing.T) {
		x1 := &app.EveCorporation{ID: 6}
		assert.False(t, x1.Equal(nil))
	})
}

func TestEveRace_FactionID(t *testing.T) {
	t.Run("known race", func(t *testing.T) {
		x := app.EveRace{ID: 1}
		got, ok := x.FactionID()
		assert.True(t, ok)
		xassert.Equal(t, 500001, got)
	})
	t.Run("unknown race", func(t *testing.T) {
		x := app.EveRace{ID: -999}
		_, ok := x.FactionID()
		assert.False(t, ok)
	})
}

func TestMembershipHistoryItem_OrganizationName(t *testing.T) {
	t.Run("has organization", func(t *testing.T) {
		x := app.MembershipHistoryItem{Organization: &app.EveEntity{Name: "Alpha"}}
		xassert.Equal(t, "Alpha", x.OrganizationName())
	})
	t.Run("no organization", func(t *testing.T) {
		x := app.MembershipHistoryItem{}
		xassert.Equal(t, "?", x.OrganizationName())
	})
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
