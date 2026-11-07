package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterWalletTransaction(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		region := location.SolarSystem.ValueOrZero().Constellation.Region
		require.NoError(t, err)
		i, err := st.GetCharacterWalletTransaction(ctx, storage.GetCharacterWalletTransactionParams{
			CharacterID:   c.ID,
			TransactionID: 42,
		})
		require.NoError(t, err)
		xassert.Equal(t, client, i.Client)
		xassert.Equal(t, date.UTC(), i.Date.UTC())
		xassert.Equal(t, eveType.ID, i.Type.ID)
		xassert.Equal(t, eveType.Name, i.Type.Name)
		assert.True(t, i.IsBuy)
		assert.True(t, i.IsPersonal)
		xassert.Equal(t, 99, i.JournalRefID)
		xassert.Equal(t, location.ID, i.Location.ID)
		xassert.Equal(t,
			&app.EveLocationShort{
				ID:             location.ID,
				Name:           optional.New(location.Name),
				SecurityStatus: i.Location.SecurityStatus,
			},
			i.Location,
		)
		xassert.Equal(t, c.ID, i.CharacterID)
		xassert.Equal(t, 7, i.Quantity)
		xassert.Equal(t, 123.45, i.UnitPrice)
		xassert.Equal(t, location.ID, i.Location.ID)
		xassert.Equal(t,
			&app.EntityShort{
				ID:   region.ID,
				Name: region.Name,
			},
			i.Region,
		)
	})
	t.Run("can list IDs of existing entries for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		e1 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction()
		// when
		got, err := st.ListCharacterWalletTransactionIDs(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(e1.TransactionID, e2.TransactionID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list existing entries for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		t1 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		t2 := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction()
		// when
		oo, err := st.ListCharacterWalletTransactions(ctx, c.ID)
		// then
		require.NoError(t, err)
		got := set.Of(xslices.Map(oo, func(x *app.CharacterWalletTransaction) int64 {
			return x.TransactionID
		})...)
		want := set.Of(t1.TransactionID, t2.TransactionID)
		xassert.Equal(t, want, got)
	})

	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetCharacterWalletTransaction(t.Context(), storage.GetCharacterWalletTransactionParams{
			CharacterID:   1,
			TransactionID: 2,
		})
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})

	t.Run("returns 0 as highest ID when a character has no transactions", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		other := factory.CreateCharacterFull()
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: other.ID})
		// when
		got, err := st.GetCharacterWalletTransactionLastID(ctx, c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, got)
	})
	t.Run("can return the highest stored ID of a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		other := factory.CreateCharacterFull()
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: other.ID, TransactionID: 9})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID, TransactionID: 7})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID, TransactionID: 5})
		// when
		got, err := st.GetCharacterWalletTransactionLastID(ctx, c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 7, got)
	})
	t.Run("returns invalid error when character is missing", func(t *testing.T) {
		// when
		_, err := st.GetCharacterWalletTransactionLastID(ctx, 0)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})

	t.Run("can create multiple transactions at once", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		arg := storage.CreateCharacterWalletTransactionParams{
			CharacterID:  c.ID,
			ClientID:     factory.CreateEveEntityCharacter().ID,
			Date:         time.Now(),
			EveTypeID:    factory.CreateEveType().ID,
			JournalRefID: 99,
			LocationID:   factory.CreateEveLocationStructure().ID,
		}
		arg1, arg2 := arg, arg
		arg1.TransactionID = 1
		arg2.TransactionID = 2
		// when
		err := st.CreateCharacterWalletTransactions(ctx, []storage.CreateCharacterWalletTransactionParams{arg1, arg2})
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterWalletTransactionIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1, 2), got)
	})
	t.Run("stores nothing when one of multiple transactions is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		arg := storage.CreateCharacterWalletTransactionParams{
			CharacterID:   c.ID,
			ClientID:      factory.CreateEveEntityCharacter().ID,
			Date:          time.Now(),
			EveTypeID:     factory.CreateEveType().ID,
			JournalRefID:  99,
			LocationID:    factory.CreateEveLocationStructure().ID,
			TransactionID: 1,
		}
		invalid := arg
		invalid.TransactionID = 0
		// when
		err := st.CreateCharacterWalletTransactions(ctx, []storage.CreateCharacterWalletTransactionParams{arg, invalid})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
		got, err := st.GetCharacterWalletTransactionLastID(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, got)
	})
}
