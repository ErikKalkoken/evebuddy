package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestEveTypeDogmaEffect(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x := factory.CreateEveType()
		arg := storage.CreateEveTypeDogmaEffectParams{
			DogmaEffectID: 42,
			EveTypeID:     x.ID,
			IsDefault:     true,
		}
		// when
		err := st.CreateEveTypeDogmaEffect(ctx, arg)
		// then
		if assert.NoError(t, err) {
			v, err := st.GetEveTypeDogmaEffect(ctx, x.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, v)
			}
		}
	})
}
