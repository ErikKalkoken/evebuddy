package app_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestEveAlliance(t *testing.T) {
	x := &app.EveAlliance{
		ID:   42,
		Name: "name",
	}
	ee := x.EveEntity()
	assert.EqualValues(t, 42, ee.ID)
	assert.EqualValues(t, "name", ee.Name)
	assert.EqualValues(t, app.EveEntityAlliance, ee.Category)
}

func TestEveCharacter_Alliance(t *testing.T) {
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

func TestEveCharacter_Description(t *testing.T) {
	x := &app.EveCharacter{Description: "alpha<br>bravo"}
	assert.Equal(t, "alpha\nbravo", x.DescriptionPlain())
}

func TestEveCharacter_Faction(t *testing.T) {
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

func TestEveCharacter_Race(t *testing.T) {
	x1 := &app.EveCharacter{Race: &app.EveRace{Description: "description"}}
	assert.Equal(t, "description", x1.RaceDescription())

	x2 := &app.EveCharacter{}
	assert.Equal(t, "", x2.RaceDescription())
}

func TestEveCharacter_EveEntity(t *testing.T) {
	x1 := &app.EveCharacter{ID: 42, Name: "name"}
	x2 := x1.EveEntity()
	assert.EqualValues(t, 42, x2.ID)
	assert.EqualValues(t, "name", x2.Name)
	assert.EqualValues(t, app.EveEntityCharacter, x2.Category)
}

func TestEveCharacter_IsIdentical(t *testing.T) {
	t.Run("should report when same", func(t *testing.T) {
		x1 := &app.EveCharacter{
			Alliance:       &app.EveEntity{ID: 1},
			Birthday:       time.Now(),
			Corporation:    &app.EveEntity{ID: 2},
			Description:    "abc",
			Faction:        &app.EveEntity{ID: 3},
			Gender:         "male",
			ID:             4,
			Name:           "Bruce Wayne",
			Race:           &app.EveRace{ID: 4},
			SecurityStatus: -4.5,
			Title:          "def",
		}
		x2 := x1
		assert.True(t, x1.Equal(x2))
	})
	t.Run("should report when not same", func(t *testing.T) {
		x1 := &app.EveCharacter{
			Alliance:       &app.EveEntity{ID: 1},
			Birthday:       time.Now(),
			Corporation:    &app.EveEntity{ID: 2},
			Description:    "abc",
			Faction:        &app.EveEntity{ID: 3},
			Gender:         "male",
			ID:             4,
			Name:           "Bruce Wayne",
			Race:           &app.EveRace{ID: 4},
			SecurityStatus: -4.5,
			Title:          "def",
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
	assert.EqualValues(t, 42, x2.ID)
	assert.EqualValues(t, "name", x2.Name)
	assert.EqualValues(t, app.EveEntityCorporation, x2.Category)
}

func TestEveCorporation_IsIdentical(t *testing.T) {
	t.Run("should report when same", func(t *testing.T) {
		x1 := app.EveCorporation{
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
			Shares:      optional.New(8),
			TaxRate:     9.1,
			Ticker:      "ghi",
			URL:         "jkl",
			WarEligible: true,
			Timestamp:   time.Now(),
		}
		x2 := x1
		assert.True(t, x1.Equal(x2))
	})
	t.Run("should report when not same", func(t *testing.T) {
		x1 := app.EveCorporation{
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
			Shares:      optional.New(8),
			TaxRate:     9.1,
			Ticker:      "ghi",
			URL:         "jkl",
			WarEligible: true,
			Timestamp:   time.Now(),
		}
		x2 := app.EveCorporation{
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
