package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCorporationWalletTransaction(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		date := time.Now()
		client := factory.CreateEveEntityCorporation()
		eveType := factory.CreateEveType()
		location := factory.CreateEveLocationStructure()
		arg := storage.CreateCorporationWalletTransactionParams{
			ClientID:      client.ID,
			Date:          date,
			DivisionID:    1,
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
		err := r.CreateCorporationWalletTransaction(ctx, arg)
		// then
		region := location.SolarSystem.Constellation.Region
		if assert.NoError(t, err) {
			i, err := r.GetCorporationWalletTransaction(ctx, storage.GetCorporationWalletTransactionParams{
				CorporationID: c.ID,
				DivisionID:    1,
				TransactionID: 42,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, client, i.Client)
				assert.Equal(t, date.UTC(), i.Date.UTC())
				assert.Equal(t, eveType.ID, i.Type.ID)
				assert.Equal(t, eveType.Name, i.Type.Name)
				assert.True(t, i.IsBuy)
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
				assert.Equal(t, c.ID, i.CorporationID)
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
	t.Run("can list IDs of existing entries for a corporation", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		t1 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		t2 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    2,
		})
		factory.CreateCorporationWalletTransaction()
		// when
		got, err := r.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		// then
		if assert.NoError(t, err) {
			want := set.Of(t1.TransactionID, t2.TransactionID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list existing entries for a corporation", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		t1 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		t2 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    2,
		})
		factory.CreateCorporationWalletTransaction()
		// when
		oo, err := r.ListCorporationWalletTransactions(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		// then
		if assert.NoError(t, err) {
			got := set.Of(xslices.Map(oo, func(x *app.CorporationWalletTransaction) int64 {
				return x.TransactionID
			})...)
			want := set.Of(t1.TransactionID, t2.TransactionID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can delete transactions", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		e1 := factory.CreateCorporationWalletTransaction()
		e2 := factory.CreateCorporationWalletTransaction()
		// when
		err := r.DeleteCorporationWalletTransactions(ctx, e1.CorporationID, app.Division(e1.DivisionID))
		// then
		if assert.NoError(t, err) {
			x1, err := r.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
				CorporationID: e1.CorporationID,
				DivisionID:    e1.DivisionID,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 0, x1.Size())
			}
			x2, err := r.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
				CorporationID: e2.CorporationID,
				DivisionID:    e2.DivisionID,
			})
			if assert.NoError(t, err) {
				assert.Greater(t, x2.Size(), 0)
			}
		}
	})
}
