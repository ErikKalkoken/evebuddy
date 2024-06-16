package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
)

func TestEveTypeDogmaAttribute(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		et := factory.CreateEveType()
		eda := factory.CreateEveDogmaAttribute()
		arg := sqlite.CreateEveTypeDogmaAttributeParams{
			DogmaAttributeID: eda.ID,
			EveTypeID:        et.ID,
			Value:            123.45,
		}
		// when
		err := r.CreateEveTypeDogmaAttribute(ctx, arg)
		// then
		if assert.NoError(t, err) {
			v, err := r.GetEveTypeDogmaAttribute(ctx, et.ID, eda.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, float32(123.45), v)
			}
		}
	})
}
