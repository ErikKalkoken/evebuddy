package corporationservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetWalletBalance(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	s := corporationservice.NewFake(st)
	b := factory.CreateCorporationWalletBalance()
	t.Run("return existing balance", func(t *testing.T) {
		got, err := s.GetWalletBalance(ctx, b.CorporationID, app.Division(b.DivisionID))
		if assert.NoError(t, err) {
			assert.EqualValues(t, b.Balance, got)
		}
	})
	t.Run("return not found error", func(t *testing.T) {
		_, err := s.GetWalletBalance(ctx, b.CorporationID, 99)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
