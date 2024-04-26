package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"example/evebuddy/internal/testutil"
)

func TestEveCharacter(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corp := factory.CreateEveEntityCorporation()
		race := factory.CreateEveRace()
		c := model.EveCharacter{ID: 1, Name: "Erik", Corporation: corp, Race: race}
		// when
		err := r.UpdateOrCreateEveCharacter(ctx, &c)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetEveCharacter(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, c.Name, r.Name)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCharacter()
		// when
		c1.Name = "Erik"
		err := r.UpdateOrCreateEveCharacter(ctx, &c1)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetEveCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, "Erik", c2.Name)
			}
		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateEveCharacter()
		// when
		err := r.DeleteEveCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetEveCharacter(ctx, c.ID)
			assert.ErrorIs(t, err, storage.ErrNotFound)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetEveCharacter(ctx, 99)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("can fetch character by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCharacter()
		// when
		c2, err := r.GetEveCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.Birthday.Unix(), c2.Birthday.Unix())
			assert.Equal(t, c1.Corporation, c2.Corporation)
			assert.Equal(t, c1.Description, c2.Description)
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Name, c2.Name)
			assert.Equal(t, int32(0), c2.Alliance.ID)
			assert.Equal(t, int32(0), c2.Faction.ID)
		}
	})
	t.Run("can fetch character by ID with all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveCharacter()
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityFaction})
		c1 := factory.CreateEveCharacter(
			model.EveCharacter{
				Alliance: alliance,
				Faction:  faction,
			})
		// when
		c2, err := r.GetEveCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, c2.Alliance)
			assert.Equal(t, c1.Birthday.Unix(), c2.Birthday.Unix())
			assert.Equal(t, c1.Corporation, c2.Corporation)
			assert.Equal(t, c1.Description, c2.Description)
			assert.Equal(t, faction, c2.Faction)
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Name, c2.Name)
		}
	})
}
