package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestEveTypeDogmaAttribute(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x := factory.CreateEveType()
		arg := storage.CreateEveTypeDogmaAttributeParams{
			DogmaAttributeID: 42,
			EveTypeID:        x.ID,
			Value:            123.45,
		}
		// when
		err := r.CreateEveTypeDogmaAttribute(ctx, arg)
		// then
		if assert.NoError(t, err) {
			v, err := r.GetEveTypeDogmaAttribute(ctx, 42, x.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 123.45, v)
			}
		}
	})
}
