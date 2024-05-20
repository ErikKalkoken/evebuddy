package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestCharacterImplant(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterImplantParams{
			EveTypeID:   eveType.ID,
			CharacterID: c.ID,
		}
		// when
		err := r.CreateCharacterImplant(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterImplant(ctx, c.ID, arg.EveTypeID)
			if assert.NoError(t, err) {
				assert.Equal(t, eveType, x.EveType)
			}
		}
	})
	t.Run("can delete for character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		o := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		// when
		err := r.DeleteCharacterImplants(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetCharacterImplant(ctx, c.ID, o.EveType.ID)
			assert.ErrorIs(t, err, storage.ErrNotFound)
		}
	})

}
