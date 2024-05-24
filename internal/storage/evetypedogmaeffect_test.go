package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestEveTypeDogmaEffect(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x := factory.CreateEveType()
		arg := storage.CreateEveTypeDogmaEffectParams{
			DogmaEffectID: 42,
			EveTypeID:     x.ID,
			IsDefault:     true,
		}
		// when
		err := r.CreateEveTypeDogmaEffect(ctx, arg)
		// then
		if assert.NoError(t, err) {
			v, err := r.GetEveTypeDogmaEffect(ctx, x.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, v)
			}
		}
	})
}
