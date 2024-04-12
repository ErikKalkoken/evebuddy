package model_test

import (
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"

	"testing"

	"github.com/stretchr/testify/assert"
)

func mustNoError(err error) {
	if err != nil {
		panic(err)
	}
}

func TestCharacter(t *testing.T) {
	t.Run("can save new", func(t *testing.T) {
		// given
		model.TruncateTables()
		corp := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityCorporation})
		c := model.Character{ID: 1, Name: "Erik", Corporation: corp}
		// when
		err := c.Save()
		// then
		if assert.NoError(t, err) {
			r, err := model.GetCharacter(c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, c, r)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter(model.Character{Name: "Erik"})
		c.Name = "John"
		mustNoError(c.Save())
		// when
		got, err := model.GetCharacter(c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.Birthday.Unix(), got.Birthday.Unix())
			assert.Equal(t, c.Name, got.Name)
		}
	})
	t.Run("can fetch character by ID with corporation only", func(t *testing.T) {
		// given
		model.TruncateTables()
		factory.CreateCharacter()
		c := factory.CreateCharacter()
		mustNoError(c.Save())
		// when
		r, err := model.GetCharacter(c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.AllianceID, r.AllianceID)
			assert.Equal(t, c.Birthday.Unix(), r.Birthday.Unix())
			assert.Equal(t, c.Corporation, r.Corporation)
			assert.Equal(t, c.CorporationID, r.CorporationID)
			assert.Equal(t, c.Description, r.Description)
			assert.Equal(t, c.FactionID, r.FactionID)
			assert.Equal(t, c.ID, r.ID)
			assert.Equal(t, c.Name, r.Name)
		}
	})
	t.Run("can fetch character by ID with alliance and faction", func(t *testing.T) {
		// given
		model.TruncateTables()
		factory.CreateCharacter()
		alliance := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityAlliance})
		c := factory.CreateCharacter(model.Character{Alliance: alliance})
		mustNoError(c.Save())
		// when
		r, err := model.GetCharacter(c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.AllianceID, r.AllianceID)
			assert.Equal(t, c.Birthday.Unix(), r.Birthday.Unix())
			assert.Equal(t, c.Corporation, r.Corporation)
			assert.Equal(t, c.CorporationID, r.CorporationID)
			assert.Equal(t, c.Description, r.Description)
			assert.Equal(t, c.FactionID, r.FactionID)
			assert.Equal(t, c.ID, r.ID)
			assert.Equal(t, c.Name, r.Name)
		}
	})
	t.Run("can fetch character alliance from character", func(t *testing.T) {
		// given
		model.TruncateTables()
		factory.CreateCharacter()
		alliance := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityAlliance})
		c := factory.CreateCharacter(model.Character{Alliance: alliance})
		mustNoError(c.Save())
		// when
		err := c.GetAlliance()
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, c.Alliance)
		}
	})
	t.Run("can fetch faction from character", func(t *testing.T) {
		// given
		model.TruncateTables()
		factory.CreateCharacter()
		faction := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityFaction})
		c := factory.CreateCharacter(model.Character{Faction: faction})
		mustNoError(c.Save())
		// when
		err := c.GetFaction()
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, faction, c.Faction)
		}
	})
	t.Run("can fetch all", func(t *testing.T) {
		// given
		model.TruncateTables()
		c1 := factory.CreateCharacter(model.Character{Name: "Bravo"})
		c2 := factory.CreateCharacter(model.Character{Name: "Alpha"})
		mustNoError(c2.Save())
		// when
		got, err := model.ListCharacters()
		// then
		if assert.NoError(t, err) {
			assert.Len(t, got, 2)
			gotIDs := set.NewFromSlice([]int32{got[0].ID, got[1].ID})
			wantIDs := set.NewFromSlice([]int32{c2.ID, c1.ID})
			assert.Equal(t, wantIDs, gotIDs)
			assert.Equal(t, c2.Corporation.Name, got[0].Corporation.Name)
		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		// when
		err := model.DeleteCharacter(c.ID)
		// then
		if assert.NoError(t, err) {
			_, err := model.GetCharacter(c.ID)
			assert.Error(t, err)
		}
	})
}
