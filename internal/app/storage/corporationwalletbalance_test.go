package storage_test

import (
	"context"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCorporationWalletBalance(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		// when
		err := st.UpdateOrCreateCorporationWalletBalance(ctx, storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    3,
			Balance:       12.34,
		})
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCorporationWalletBalance(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    3,
			})
			if assert.NoError(t, err) {
				assert.EqualValues(t, c.ID, x.CorporationID)
				assert.EqualValues(t, 3, x.DivisionID)
				assert.EqualValues(t, 12.34, x.Balance)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateCorporationWalletBalance()
		// when
		err := st.UpdateOrCreateCorporationWalletBalance(ctx, storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: x1.CorporationID,
			DivisionID:    x1.DivisionID,
			Balance:       12.34,
		})
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCorporationWalletBalance(ctx, storage.CorporationDivision{
				CorporationID: x1.CorporationID,
				DivisionID:    x1.DivisionID,
			})
			if assert.NoError(t, err) {
				assert.EqualValues(t, 12.34, x.Balance)
			}
		}
	})
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		e2 := factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    2,
		})
		factory.CreateCorporationWalletJournalEntry()
		// when
		oo, err := st.ListCorporationWalletBalances(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CorporationWalletBalance) (int32, float64) {
				return x.DivisionID, x.Balance
			}))
			want := map[int32]float64{
				e1.DivisionID: e1.Balance,
				e2.DivisionID: e2.Balance,
			}
			assert.Equal(t, want, got)
		}
	})
	t.Run("can delete entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		e1 := factory.CreateCorporationWalletBalance()
		e2 := factory.CreateCorporationWalletBalance()
		// when
		err := st.DeleteCorporationWalletBalance(ctx, e1.CorporationID)
		// then
		if assert.NoError(t, err) {
			x1, err := st.ListCorporationWalletBalances(ctx, e1.CorporationID)
			if assert.NoError(t, err) {
				assert.Equal(t, 0, len(x1))
			}
			x2, err := st.ListCorporationWalletBalances(ctx, e2.CorporationID)
			if assert.NoError(t, err) {
				assert.Greater(t, len(x2), 0)
			}
		}
	})
}
