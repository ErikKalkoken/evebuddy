package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestEveTypeDogmaAttribute(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		et := factory.CreateEveType()
		eda := factory.CreateEveDogmaAttribute()
		arg := storage.CreateEveTypeDogmaAttributeParams{
			DogmaAttributeID: eda.ID,
			EveTypeID:        et.ID,
			Value:            123.45,
		}
		// when
		err := st.CreateEveTypeDogmaAttribute(ctx, arg)
		// then
		if assert.NoError(t, err) {
			v, err := st.GetEveTypeDogmaAttribute(ctx, et.ID, eda.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, float32(123.45), v)
			}
		}
	})
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		et := factory.CreateEveType()
		o1 := factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID: et.ID,
		})
		o2 := factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID: et.ID,
		})
		factory.CreateEveTypeDogmaAttribute()
		// when
		oo, err := st.ListEveTypeDogmaAttributesForType(ctx, et.ID)
		// then
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, []*app.EveTypeDogmaAttribute{o1, o2}, oo)
		}
	})
}
