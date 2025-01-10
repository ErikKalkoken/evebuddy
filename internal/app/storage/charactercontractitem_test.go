package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCharacterContractItem(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterContract()
		et := factory.CreateEveType()
		arg := storage.CreateCharacterContractItemParams{
			ContractID:  c.ID,
			IsIncluded:  true,
			IsSingleton: true,
			Quantity:    7,
			RawQuantity: -5,
			RecordID:    42,
			TypeID:      et.ID,
		}
		// when
		err := r.CreateCharacterContractItem(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterContractItem(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsIncluded)
				assert.True(t, o.IsSingleton)
				assert.Equal(t, 7, o.Quantity)
				assert.Equal(t, -5, o.RawQuantity)
				assert.Equal(t, et, o.Type)
			}
		}
	})
	t.Run("can list existing items", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterContract()
		factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{ContractID: c.ID})
		factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{ContractID: c.ID})
		factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{ContractID: c.ID})
		// when
		oo, err := r.ListCharacterContractItems(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 3)
		}
	})
}
