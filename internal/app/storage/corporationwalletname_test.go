package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestCorporationWalletName(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		// when
		err := r.UpdateOrCreateCorporationWalletName(ctx, storage.UpdateOrCreateCorporationWalletNameParams{
			CorporationID: c.ID,
			DivisionID:    3,
			Name:          "Alpha",
		})
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCorporationWalletName(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    3,
			})
			if assert.NoError(t, err) {
				assert.EqualValues(t, c.ID, x.CorporationID)
				assert.EqualValues(t, 3, x.DivisionID)
				assert.EqualValues(t, "Alpha", x.Name)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCorporationWalletName()
		// when
		err := r.UpdateOrCreateCorporationWalletName(ctx, storage.UpdateOrCreateCorporationWalletNameParams{
			CorporationID: x1.CorporationID,
			DivisionID:    x1.DivisionID,
			Name:          "Alpha",
		})
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCorporationWalletName(ctx, storage.CorporationDivision{
				CorporationID: x1.CorporationID,
				DivisionID:    x1.DivisionID,
			})
			if assert.NoError(t, err) {
				assert.EqualValues(t, "Alpha", x.Name)
			}
		}
	})
}
