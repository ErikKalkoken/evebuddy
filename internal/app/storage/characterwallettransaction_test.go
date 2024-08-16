package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestWalletTransaction(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		date := time.Now()
		client := factory.CreateEveEntityCharacter()
		eveType := factory.CreateEveType()
		location := factory.CreateLocationStructure()
		arg := storage.CreateCharacterWalletTransactionParams{
			ClientID:      client.ID,
			Date:          date,
			EveTypeID:     eveType.ID,
			IsBuy:         true,
			IsPersonal:    true,
			JournalRefID:  99,
			LocationID:    location.ID,
			CharacterID:   c.ID,
			Quantity:      7,
			UnitPrice:     123.45,
			TransactionID: 42,
		}
		// when
		err := r.CreateCharacterWalletTransaction(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetCharacterWalletTransaction(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, client, i.Client)
				assert.Equal(t, date.UTC(), i.Date.UTC())
				assert.Equal(t, eveType.ID, i.EveType.ID)
				assert.Equal(t, eveType.Name, i.EveType.Name)
				assert.True(t, i.IsBuy)
				assert.True(t, i.IsPersonal)
				assert.Equal(t, int64(99), i.JournalRefID)
				assert.Equal(t, location.ID, i.Location.ID)
				assert.Equal(t, location.Name, i.Location.Name)
				assert.Equal(t, c.ID, i.CharacterID)
				assert.Equal(t, int32(7), i.Quantity)
				assert.Equal(t, 123.45, i.UnitPrice)
			}
		}
	})
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		e3 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		// when
		ids, err := r.ListCharacterWalletTransactionIDs(ctx, c.ID)
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
		c := factory.CreateCharacter()
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		// when
		ee, err := r.ListCharacterWalletTransactions(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ee, 3)
		}
	})
}
