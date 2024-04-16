package repository_test

import (
	"context"
	"example/evebuddy/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCharacter(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		corp := factory.CreateEveEntityCorporation()
		c := repository.Character{ID: 1, Name: "Erik", Corporation: corp}
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
		repository.TruncateTables(db)
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
		repository.TruncateTables(db)
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
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		err := r.DeleteCharacter(ctx, &c)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetCharacter(ctx, c.ID)
			assert.Error(t, err)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		_, err := r.GetCharacter(ctx, 99)
		// then
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}
