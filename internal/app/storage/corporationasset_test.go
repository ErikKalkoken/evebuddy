package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCorporationAsset(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		eveType := factory.CreateEveType()
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID: eveType.ID, AveragePrice: 1.24,
		})
		// when
		err := st.CreateCorporationAsset(ctx, storage.CreateCorporationAssetParams{
			CorporationID:   c.ID,
			EveTypeID:       eveType.ID,
			IsBlueprintCopy: false,
			IsSingleton:     true,
			ItemID:          42,
			LocationFlag:    app.FlagHangar,
			LocationID:      99,
			LocationType:    app.TypeOther,
			Name:            "Alpha",
			Quantity:        7,
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCorporationAsset(ctx, c.ID, 42)
		require.NoError(t, err)
		assert.Equal(t, eveType.ID, x.Type.ID)
		assert.Equal(t, eveType.Name, x.Type.Name)
		assert.False(t, x.IsBlueprintCopy)
		assert.True(t, x.IsSingleton)
		assert.Equal(t, int64(42), x.ItemID)
		assert.Equal(t, app.FlagHangar, x.LocationFlag)
		assert.Equal(t, int64(99), x.LocationID)
		assert.Equal(t, app.TypeOther, x.LocationType)
		assert.Equal(t, "Alpha", x.Name)
		assert.EqualValues(t, 7, x.Quantity)
		assert.Equal(t, 1.24, x.Price.ValueOrZero())
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		x1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		// when
		err := st.UpdateCorporationAsset(ctx, storage.UpdateCorporationAssetParams{
			CorporationID: c.ID,
			ItemID:        x1.ItemID,
			LocationFlag:  app.FlagHangar,
			LocationID:    99,
			LocationType:  app.TypeOther,
			Quantity:      7,
		})
		// then
		require.NoError(t, err)
		x2, err := st.GetCorporationAsset(ctx, c.ID, x1.ItemID)
		require.NoError(t, err)
		assert.Equal(t, app.FlagHangar, x2.LocationFlag)
		assert.Equal(t, int64(99), x2.LocationID)
		assert.Equal(t, app.TypeOther, x2.LocationType)
		assert.Equal(t, x1.Name, x2.Name)
		assert.EqualValues(t, 7, x2.Quantity)
	})
	t.Run("can update name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		x1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		// when
		err := st.UpdateCorporationAssetName(ctx, storage.UpdateCorporationAssetNameParams{
			CorporationID: c.ID,
			ItemID:        x1.ItemID,
			Name:          "Alpha",
		})
		// then
		require.NoError(t, err)
		x2, err := st.GetCorporationAsset(ctx, c.ID, x1.ItemID)
		require.NoError(t, err)
		assert.Equal(t, "Alpha", x2.Name)
	})
	t.Run("can delete assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		x1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		x2 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		// when
		err := st.DeleteCorporationAssets(ctx, c.ID, set.Of(x2.ItemID))
		// then
		require.NoError(t, err)
		got, err := st.ListCorporationAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(x1.ItemID)
		xassert.EqualSet(t, want, got)
	})
	t.Run("can list assets for corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		ca1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		ca2 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		// when
		got, err := st.ListCorporationAssets(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := []*app.CorporationAsset{ca1, ca2}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can list all assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		ca1 := factory.CreateCorporationAsset()
		ca2 := factory.CreateCorporationAsset()
		// when
		got, err := st.ListAllCorporationAssets(ctx)
		// then
		require.NoError(t, err)
		want := []*app.CorporationAsset{ca1, ca2}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can calculate total asset value for corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		ca1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			Quantity:      1,
		})
		ca2 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			Quantity:      2,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca1.Type.ID,
			AveragePrice: 100.1,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca2.Type.ID,
			AveragePrice: 200.2,
		})
		// when
		got, err := st.CalculateCorporationAssetTotalValue(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.InDelta(t, 500.5, got, 0.1)
	})
	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetCorporationAsset(ctx, 1, 2)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
