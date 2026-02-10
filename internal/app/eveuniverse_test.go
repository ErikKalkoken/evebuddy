package app_test

import (
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
	ee := x.EveEntity()
	xassert.Equal(t, 42, ee.ID)
	xassert.Equal(t, "name", ee.Name)
	xassert.Equal(t, app.EveEntityAlliance, ee.Category)
}

func TestEveCharacter_Alliance(t *testing.T) {
	x1 := &app.EveCharacter{
		Alliance: &app.EveEntity{
			Name: "alliance",
		},
	}
	xassert.Equal(t, "alliance", x1.AllianceName())
	assert.True(t, x1.HasAlliance())

	x2 := &app.EveCharacter{}
	xassert.Equal(t, "", x2.AllianceName())
	assert.False(t, x2.HasAlliance())
}

func TestEveCharacter_Description(t *testing.T) {
	x := &app.EveCharacter{
		Description: optional.New("alpha<br>bravo"),
	}
	xassert.Equal(t, "alpha\nbravo", x.DescriptionPlain())
}

func TestEveCharacter_Faction(t *testing.T) {
	x1 := &app.EveCharacter{
		Faction: &app.EveEntity{
			Name: "faction",
		},
	}
	xassert.Equal(t, "faction", x1.FactionName())
	assert.True(t, x1.HasFaction())

	x2 := &app.EveCharacter{}
	xassert.Equal(t, "", x2.FactionName())
	assert.False(t, x2.HasFaction())
}

func TestEveCharacter_Race(t *testing.T) {
	x1 := &app.EveCharacter{Race: &app.EveRace{Description: "description"}}
	xassert.Equal(t, "description", x1.RaceDescription())

	x2 := &app.EveCharacter{}
	xassert.Equal(t, "", x2.RaceDescription())
}

func TestEveCharacter_EveEntity(t *testing.T) {
	x1 := &app.EveCharacter{ID: 42, Name: "name"}
	x2 := x1.EveEntity()
	xassert.Equal(t, 42, x2.ID)
	xassert.Equal(t, "name", x2.Name)
	xassert.Equal(t, app.EveEntityCharacter, x2.Category)
}

func TestEveCharacter_IsIdentical(t *testing.T) {
	t.Run("should report when same", func(t *testing.T) {
		x1 := &app.EveCharacter{
			Alliance:       &app.EveEntity{ID: 1},
			Birthday:       time.Now(),
			Corporation:    &app.EveEntity{ID: 2},
			Description:    optional.New("abc"),
			Faction:        &app.EveEntity{ID: 3},
			Gender:         "male",
			ID:             4,
			Name:           "Bruce Wayne",
			Race:           &app.EveRace{ID: 4},
			SecurityStatus: optional.New(-4.5),
			Title:          optional.New("def"),
		}
		x2 := x1
		assert.True(t, x1.Equal(x2))
	})
	t.Run("should report when not same", func(t *testing.T) {
		x1 := &app.EveCharacter{
			Alliance:       &app.EveEntity{ID: 1},
			Birthday:       time.Now(),
			Corporation:    &app.EveEntity{ID: 2},
			Description:    optional.New("abc"),
			Faction:        &app.EveEntity{ID: 3},
			Gender:         "male",
			ID:             4,
			Name:           "Bruce Wayne",
			Race:           &app.EveRace{ID: 4},
			SecurityStatus: optional.New(-4.5),
			Title:          optional.New("def"),
		}
		x2 := &app.EveCharacter{
			ID: 4,
		}
		assert.False(t, x1.Equal(x2))
	})
}

func TestEveCorporation_Alliance(t *testing.T) {
	x1 := &app.EveCorporation{Alliance: &app.EveEntity{}}
	assert.True(t, x1.HasAlliance())

	x2 := &app.EveCorporation{}
	assert.False(t, x2.HasAlliance())
}

func TestEveCorporation_Faction(t *testing.T) {
	x1 := &app.EveCorporation{Faction: &app.EveEntity{}}
	assert.True(t, x1.HasFaction())

	x2 := &app.EveCorporation{}
	assert.False(t, x2.HasFaction())
}

func TestEveCorporation_EveEntity(t *testing.T) {
	x1 := &app.EveCorporation{ID: 42, Name: "name"}
	x2 := x1.EveEntity()
	xassert.Equal(t, 42, x2.ID)
	xassert.Equal(t, "name", x2.Name)
	xassert.Equal(t, app.EveEntityCorporation, x2.Category)
}

func TestEveCorporation_IsIdentical(t *testing.T) {
	t.Run("should report when same", func(t *testing.T) {
		x1 := &app.EveCorporation{
			Alliance:    &app.EveEntity{ID: 1},
			Ceo:         &app.EveEntity{ID: 2},
			Creator:     &app.EveEntity{ID: 3},
			DateFounded: optional.New(time.Now().Add(-3 * time.Hour)),
			Description: "abc",
			Faction:     &app.EveEntity{ID: 4},
			HomeStation: &app.EveEntity{ID: 5},
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
			Alliance:    &app.EveEntity{ID: 1},
			Ceo:         &app.EveEntity{ID: 2},
			Creator:     &app.EveEntity{ID: 3},
			DateFounded: optional.New(time.Now().Add(-3 * time.Hour)),
			Description: "abc",
			Faction:     &app.EveEntity{ID: 4},
			HomeStation: &app.EveEntity{ID: 5},
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
