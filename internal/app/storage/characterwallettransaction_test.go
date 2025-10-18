package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/stretchr/testify/assert"
)

func TestCharacterWalletTransaction(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		date := time.Now()
		client := factory.CreateEveEntityCharacter()
		eveType := factory.CreateEveType()
		location := factory.CreateEveLocationStructure()
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
		err := st.CreateCharacterWalletTransaction(ctx, arg)
		// then
		region := location.SolarSystem.Constellation.Region
		if assert.NoError(t, err) {
			i, err := st.GetCharacterWalletTransaction(ctx, storage.GetCharacterWalletTransactionParams{
				CharacterID:   c.ID,
				TransactionID: 42,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, client, i.Client)
				assert.Equal(t, date.UTC(), i.Date.UTC())
				assert.Equal(t, eveType.ID, i.Type.ID)
				assert.Equal(t, eveType.Name, i.Type.Name)
				assert.True(t, i.IsBuy)
				assert.True(t, i.IsPersonal)
				assert.Equal(t, int64(99), i.JournalRefID)
				assert.Equal(t, location.ID, i.Location.ID)
				assert.Equal(t,
					&app.EveLocationShort{
						ID:             location.ID,
						Name:           optional.New(location.Name),
						SecurityStatus: i.Location.SecurityStatus,
					},
					i.Location,
				)
				assert.Equal(t, c.ID, i.CharacterID)
				assert.Equal(t, int32(7), i.Quantity)
				assert.Equal(t, 123.45, i.UnitPrice)
				assert.Equal(t, location.ID, i.Location.ID)
				assert.Equal(t,
					&app.EntityShort[int32]{
						ID:   region.ID,
						Name: region.Name,
					},
					i.Region,
				)
			}
		}
	})
	t.Run("can list IDs of existing entries for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		e1 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction()
		// when
		got, err := st.ListCharacterWalletTransactionIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(e1.TransactionID, e2.TransactionID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list existing entries for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		t1 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		t2 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction()
		// when
		oo, err := st.ListCharacterWalletTransactions(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Of(xslices.Map(oo, func(x *app.CharacterWalletTransaction) int64 {
				return x.TransactionID
			})...)
			want := set.Of(t1.TransactionID, t2.TransactionID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}
