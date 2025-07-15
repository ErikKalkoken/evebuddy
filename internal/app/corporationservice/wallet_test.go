package corporationservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetWalletBalance(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := corporationservice.NewFake(st)
	b := factory.CreateCorporationWalletBalance()
	t.Run("return existing balance", func(t *testing.T) {

		// when
		got, err := s.GetWalletBalance(ctx, b.CorporationID, app.Division(b.DivisionID))
		// then
		if assert.NoError(t, err) {
			assert.EqualValues(t, b.Balance, got)
		}
	})
	t.Run("return not found error", func(t *testing.T) {
		// when
		_, err := s.GetWalletBalance(ctx, b.CorporationID, 99)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestWalletBalancesTotal(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	t.Run("return existing balance", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		s := corporationservice.NewFake(st)
		c := factory.CreateCorporation()
		factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    app.Division1.ID(),
			Balance:       12,
		})
		factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    app.Division2.ID(),
			Balance:       24,
		})
		// when
		got, err := s.GetWalletBalancesTotal(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.EqualValues(t, 36, got.ValueOrZero())
		}
	})
	t.Run("return empty when no balances found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		s := corporationservice.NewFake(st)
		c := factory.CreateCorporation()
		// when
		got, err := s.GetWalletBalancesTotal(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, got.IsEmpty())
		}
	})
}
