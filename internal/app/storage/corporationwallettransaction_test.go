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

func TestCorporationWalletTransaction(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		date := time.Now()
		client := factory.CreateEveEntityCorporation()
		eveType := factory.CreateEveType()
		location := factory.CreateEveLocationStructure()
		arg := storage.CreateCorporationWalletTransactionParams{
			ClientID:      client.ID,
			Date:          date,
			Division:      app.Division1,
			EveTypeID:     eveType.ID,
			IsBuy:         true,
			JournalRefID:  99,
			LocationID:    location.ID,
			CorporationID: c.ID,
			Quantity:      7,
			UnitPrice:     123.45,
			TransactionID: 42,
		}
		// when
		err := st.CreateCorporationWalletTransaction(ctx, arg)
		// then
		region := location.SolarSystem.ValueOrZero().Constellation.Region
		require.NoError(t, err)
		i, err := st.GetCorporationWalletTransaction(ctx, storage.GetCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division1,
			TransactionID: 42,
		})
		require.NoError(t, err)
		xassert.Equal(t, client, i.Client)
		xassert.Equal(t, date.UTC(), i.Date.UTC())
		xassert.Equal(t, eveType.ID, i.Type.ID)
		xassert.Equal(t, eveType.Name, i.Type.Name)
		assert.True(t, i.IsBuy)
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
		xassert.Equal(t, c.ID, i.CorporationID)
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
	t.Run("can list IDs of existing entries for a corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		t1 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		t2 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division2,
		})
		factory.CreateCorporationWalletTransaction()
		// when
		got, err := st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		// then
		require.NoError(t, err)
		want := set.Of(t1.TransactionID, t2.TransactionID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list existing entries for a corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		t1 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		t2 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division2,
		})
		factory.CreateCorporationWalletTransaction()
		// when
		oo, err := st.ListCorporationWalletTransactions(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		// then
		require.NoError(t, err)
		got := set.Of(xslices.Map(oo, func(x *app.CorporationWalletTransaction) int64 {
			return x.TransactionID
		})...)
		want := set.Of(t1.TransactionID, t2.TransactionID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list existing entries for a corporation for all divisions", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		t1 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		t2 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		t3 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      app.Division2,
		})
		factory.CreateCorporationWalletTransaction()
		// when
		oo, err := st.ListCorporationWalletTransactionsAll(ctx, c.ID)
		// then
		require.NoError(t, err)
		got := set.Of(xslices.Map(oo, func(x *app.CorporationWalletTransaction) int64 {
			return x.TransactionID
		})...)
		want := set.Of(t1.TransactionID, t2.TransactionID, t3.TransactionID)
		xassert.Equal(t, want, got)
	})
	t.Run("can delete transactions", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		e1 := factory.CreateCorporationWalletTransaction()
		e2 := factory.CreateCorporationWalletTransaction()
		// when
		err := st.DeleteCorporationWalletTransactions(ctx, e1.CorporationID, e1.Division)
		// then
		require.NoError(t, err)
		x1, err := st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
			CorporationID: e1.CorporationID,
			Division:      e1.Division,
		})
		require.NoError(t, err)
		xassert.Equal(t, 0, x1.Size())
		x2, err := st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
			CorporationID: e2.CorporationID,
			Division:      e2.Division,
		})
		require.NoError(t, err)
		assert.Greater(t, x2.Size(), 0)
	})

	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetCorporationWalletTransaction(t.Context(), storage.GetCorporationWalletTransactionParams{
			CorporationID: 1,
			Division:      app.Division1,
			TransactionID: 2,
		})
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})

	t.Run("can return the highest stored ID of a corporation division", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		other := factory.CreateCorporation()
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{CorporationID: other.ID, Division: app.Division1})
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{CorporationID: c.ID, Division: app.Division2})
		arg := storage.CorporationDivision{CorporationID: c.ID, Division: app.Division1}
		// when
		got, err := st.GetCorporationWalletLastTransactionID(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, got)
		// when
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{CorporationID: c.ID, Division: app.Division1, TransactionID: 7})
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{CorporationID: c.ID, Division: app.Division1, TransactionID: 5})
		got, err = st.GetCorporationWalletLastTransactionID(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 7, got)
	})
	t.Run("returns invalid error when division is missing", func(t *testing.T) {
		_, err := st.GetCorporationWalletLastTransactionID(ctx, storage.CorporationDivision{CorporationID: 1})
		assert.ErrorIs(t, err, app.ErrInvalid)
	})

	t.Run("can create multiple transactions at once", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		arg := storage.CreateCorporationWalletTransactionParams{
			ClientID:      factory.CreateEveEntityCorporation().ID,
			CorporationID: c.ID,
			Date:          time.Now(),
			Division:      app.Division1,
			EveTypeID:     factory.CreateEveType().ID,
			JournalRefID:  99,
			LocationID:    factory.CreateEveLocationStructure().ID,
		}
		arg1, arg2 := arg, arg
		arg1.TransactionID = 1
		arg2.TransactionID = 2
		// when
		err := st.CreateCorporationWalletTransactions(ctx, []storage.CreateCorporationWalletTransactionParams{arg1, arg2})
		// then
		require.NoError(t, err)
		got, err := st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1, 2), got)
	})
	t.Run("stores nothing when one of multiple transactions is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		arg := storage.CreateCorporationWalletTransactionParams{
			ClientID:      factory.CreateEveEntityCorporation().ID,
			CorporationID: c.ID,
			Date:          time.Now(),
			Division:      app.Division1,
			EveTypeID:     factory.CreateEveType().ID,
			JournalRefID:  99,
			LocationID:    factory.CreateEveLocationStructure().ID,
			TransactionID: 1,
		}
		invalid := arg
		invalid.TransactionID = 0
		// when
		err := st.CreateCorporationWalletTransactions(ctx, []storage.CreateCorporationWalletTransactionParams{arg, invalid})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
		got, err := st.GetCorporationWalletLastTransactionID(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			Division:      app.Division1,
		})
		require.NoError(t, err)
		xassert.Equal(t, 0, got)
	})
}
