package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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
