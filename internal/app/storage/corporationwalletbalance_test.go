package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestCorporationWalletBalance(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		// when
		err := r.UpdateOrCreateCorporationWalletBalance(ctx, storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    3,
			Balance:       12.34,
		})
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCorporationWalletBalance(ctx, storage.CorporationDivision{
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
		testutil.TruncateTables(db)
		x1 := factory.CreateCorporationWalletBalance()
		// when
		err := r.UpdateOrCreateCorporationWalletBalance(ctx, storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: x1.CorporationID,
			DivisionID:    x1.DivisionID,
			Balance:       12.34,
		})
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCorporationWalletBalance(ctx, storage.CorporationDivision{
				CorporationID: x1.CorporationID,
				DivisionID:    x1.DivisionID,
			})
			if assert.NoError(t, err) {
				assert.EqualValues(t, 12.34, x.Balance)
			}
		}
	})
}
