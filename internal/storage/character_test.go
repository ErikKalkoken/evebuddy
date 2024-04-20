package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"example/evebuddy/internal/testutil"
)

func TestCharacter(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corp := factory.CreateEveEntityCorporation()
		race := factory.CreateRace()
		system := factory.CreateEveEntitySolarSystem()
		c := model.Character{ID: 1, Name: "Erik", Corporation: corp, Race: race, SolarSystem: system}
		// when
		err := r.UpdateOrCreateCharacter(ctx, &c)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetCharacter(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, c.Name, r.Name)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		c1.Name = "Erik"
		err := r.UpdateOrCreateCharacter(ctx, &c1)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, "Erik", c2.Name)
			}
		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		err := r.DeleteCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetCharacter(ctx, c.ID)
			assert.ErrorIs(t, err, storage.ErrNotFound)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetCharacter(ctx, 99)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("should return first character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		factory.CreateCharacter()
		// when
		c2, err := r.GetFirstCharacter(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.ID, c2.ID)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetFirstCharacter(ctx)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("can fetch character by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		c2, err := r.GetCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.Birthday.Unix(), c2.Birthday.Unix())
			assert.Equal(t, c1.Corporation, c2.Corporation)
			assert.Equal(t, c1.Description, c2.Description)
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Name, c2.Name)
			assert.Equal(t, c1.SolarSystem, c2.SolarSystem)
			assert.Equal(t, int32(0), c2.Alliance.ID)
			assert.Equal(t, int32(0), c2.Faction.ID)
		}
	})
	t.Run("can fetch character by ID with all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateCharacter()
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity()
		system := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntitySolarSystem})
		c1 := factory.CreateCharacter(
			model.Character{
				Alliance:    alliance,
				Faction:     faction,
				SolarSystem: system,
			})
		// when
		c2, err := r.GetCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, c2.Alliance)
			assert.Equal(t, c1.Birthday.Unix(), c2.Birthday.Unix())
			assert.Equal(t, c1.Corporation, c2.Corporation)
			assert.Equal(t, c1.Description, c2.Description)
			assert.Equal(t, faction, c2.Faction)
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Name, c2.Name)
			assert.Equal(t, c1.SkillPoints, c2.SkillPoints)
			assert.Equal(t, c1.SolarSystem, c2.SolarSystem)
			assert.Equal(t, c1.WalletBalance, c2.WalletBalance)
		}
	})
}

func TestCharacterList(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("listed characters have all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity()
		system := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntitySolarSystem})
		c1 := factory.CreateCharacter(
			model.Character{
				Alliance:      alliance,
				Faction:       faction,
				SolarSystem:   system,
				SkillPoints:   1234567,
				WalletBalance: 12345.67,
			})
		// when
		cc, err := r.ListCharacters(ctx)
		// then
		if assert.NoError(t, err) {
			c2 := cc[0]
			assert.Len(t, cc, 1)
			assert.Equal(t, alliance, c2.Alliance)
			assert.Equal(t, c1.Birthday.Unix(), c2.Birthday.Unix())
			assert.Equal(t, c1.Corporation, c2.Corporation)
			assert.Equal(t, c1.Description, c2.Description)
			assert.Equal(t, faction, c2.Faction)
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Name, c2.Name)
			assert.Equal(t, c1.SkillPoints, c2.SkillPoints)
			assert.Equal(t, c1.SolarSystem, c2.SolarSystem)
			assert.Equal(t, c1.WalletBalance, c2.WalletBalance)
		}
	})
	t.Run("can list characters", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateCharacter()
		factory.CreateCharacter()
		// when
		cc, err := r.ListCharacters(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, cc, 2)
		}
	})

}
