package corporationservice_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestGetWalletBalance(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationService(corporationservice.Params{Storage: st})
	t.Run("return existing balance", func(t *testing.T) {
		// when
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		b := factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		got, err := s.GetWalletBalance(ctx, b.CorporationID, app.Division(b.DivisionID))
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, b.Balance, got)
		}
	})
	t.Run("return not found error", func(t *testing.T) {
		// when
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		b := factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		// when
		_, err := s.GetWalletBalance(ctx, b.CorporationID, 99)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestWalletBalancesTotal(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationService(corporationservice.Params{Storage: st})
	t.Run("return existing balance", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
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
			xassert.Equal(t, 36, got.ValueOrZero())
		}
	})
	t.Run("return empty when no balances found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		// when
		got, err := s.GetWalletBalancesTotal(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, got.IsEmpty())
		}
	})
}
