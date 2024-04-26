package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"example/evebuddy/internal/testutil"
)

func TestMyCharacter(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character := factory.CreateEveCharacter()
		system := factory.CreateEveSolarSystem()
		ship := factory.CreateEveType()
		c := model.MyCharacter{ID: 1, Location: system, Ship: ship, Character: character}
		// when
		err := r.UpdateOrCreateMyCharacter(ctx, &c)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetMyCharacter(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, c.Location, r.Location)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		// when
		newLocation := factory.CreateEveSolarSystem()
		c1.Location = newLocation
		newShip := factory.CreateEveType()
		c1.Ship = newShip
		err := r.UpdateOrCreateMyCharacter(ctx, &c1)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetMyCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, newLocation, c2.Location)
				assert.Equal(t, newShip, c2.Ship)
			}
		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		// when
		err := r.DeleteMyCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetMyCharacter(ctx, c.ID)
			assert.ErrorIs(t, err, storage.ErrNotFound)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetMyCharacter(ctx, 99)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("should return first character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		factory.CreateMyCharacter()
		// when
		c2, err := r.GetFirstMyCharacter(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.ID, c2.ID)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetFirstMyCharacter(ctx)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("can fetch character by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		// when
		c2, err := r.GetMyCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Location, c2.Location)
		}
	})
}

func TestMyCharacterList(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("listed characters have all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		// when
		cc, err := r.ListMyCharacters(ctx)
		// then
		if assert.NoError(t, err) {
			c2 := cc[0]
			assert.Len(t, cc, 1)
			assert.Equal(t, c1.ID, c2.ID)
		}
	})
	t.Run("can list characters", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateMyCharacter()
		factory.CreateMyCharacter()
		// when
		cc, err := r.ListMyCharacters(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, cc, 2)
		}
	})

}
