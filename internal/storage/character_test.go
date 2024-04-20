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
		c := model.Character{ID: 1, Name: "Erik", Corporation: corp}
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
		c := factory.CreateCharacter()
		// when
		c.Name = "Erik"
		err := r.UpdateOrCreateCharacter(ctx, &c)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetCharacter(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, "Erik", r.Name)
			}
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
		r, err := r.GetFirstCharacter(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.ID, r.ID)
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
		c := factory.CreateCharacter()
		// when
		r, err := r.GetCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.Birthday.Unix(), r.Birthday.Unix())
			assert.Equal(t, c.Corporation, r.Corporation)
			assert.Equal(t, c.Description, r.Description)
			assert.Equal(t, c.ID, r.ID)
			assert.Equal(t, c.Name, r.Name)
			assert.Equal(t, int32(0), r.Alliance.ID)
			assert.Equal(t, int32(0), r.Faction.ID)
			assert.Equal(t, int32(0), r.SolarSystem.ID)
		}
	})
	t.Run("can fetch character by ID with all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateCharacter()
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity()
		system := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntitySolarSystem})
		c := factory.CreateCharacter(
			model.Character{
				Alliance:      alliance,
				Faction:       faction,
				SolarSystem:   system,
				SkillPoints:   1234567,
				WalletBalance: 12345.67,
			})
		// when
		r, err := r.GetCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, r.Alliance)
			assert.Equal(t, c.Birthday.Unix(), r.Birthday.Unix())
			assert.Equal(t, c.Corporation, r.Corporation)
			assert.Equal(t, c.Description, r.Description)
			assert.Equal(t, faction, r.Faction)
			assert.Equal(t, c.ID, r.ID)
			assert.Equal(t, c.Name, r.Name)
			assert.Equal(t, c.SkillPoints, r.SkillPoints)
			assert.Equal(t, c.SolarSystem, r.SolarSystem)
			assert.Equal(t, c.WalletBalance, r.WalletBalance)
		}
	})
}
