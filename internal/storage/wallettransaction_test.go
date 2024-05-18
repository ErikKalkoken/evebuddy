package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestWalletTransaction(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		date := time.Now()
		client := factory.CreateEveEntityCharacter()
		eveType := factory.CreateEveType()
		location := factory.CreateLocationStructure()
		arg := storage.CreateWalletTransactionParams{
			ClientID:      client.ID,
			Date:          date,
			EveTypeID:     eveType.ID,
			IsBuy:         true,
			IsPersonal:    true,
			JournalRefID:  99,
			LocationID:    location.ID,
			MyCharacterID: c.ID,
			Quantity:      7,
			UnitPrice:     123.45,
			TransactionID: 42,
		}
		// when
		err := r.CreateWalletTransaction(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetWalletTransaction(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, client, i.Client)
				assert.Equal(t, date.UTC(), i.Date.UTC())
				assert.Equal(t, eveType.ID, i.EveTypeID)
				assert.Equal(t, eveType.Name, i.EveTypeName)
				assert.True(t, i.IsBuy)
				assert.True(t, i.IsPersonal)
				assert.Equal(t, int64(99), i.JournalRefID)
				assert.Equal(t, location.ID, i.LocationID)
				assert.Equal(t, location.Name, i.LocationName)
				assert.Equal(t, c.ID, i.MyCharacterID)
				assert.Equal(t, int32(7), i.Quantity)
				assert.Equal(t, 123.45, i.UnitPrice)
			}
		}
	})
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		e1 := factory.CreateWalletTransaction(storage.CreateWalletTransactionParams{MyCharacterID: c.ID})
		e2 := factory.CreateWalletTransaction(storage.CreateWalletTransactionParams{MyCharacterID: c.ID})
		e3 := factory.CreateWalletTransaction(storage.CreateWalletTransactionParams{MyCharacterID: c.ID})
		// when
		ids, err := r.ListWalletTransactionIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.NewFromSlice(ids)
			want := set.NewFromSlice([]int64{e1.TransactionID, e2.TransactionID, e3.TransactionID})
			assert.Equal(t, want, got)
		}
	})
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		factory.CreateWalletTransaction(storage.CreateWalletTransactionParams{MyCharacterID: c.ID})
		factory.CreateWalletTransaction(storage.CreateWalletTransactionParams{MyCharacterID: c.ID})
		factory.CreateWalletTransaction(storage.CreateWalletTransactionParams{MyCharacterID: c.ID})
		// when
		ee, err := r.ListWalletTransactions(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ee, 3)
		}
	})
}
