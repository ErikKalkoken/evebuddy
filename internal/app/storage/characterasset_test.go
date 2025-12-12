package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterAsset(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		eveType := factory.CreateEveType()
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID: eveType.ID, AveragePrice: 1.24,
		})
		arg := storage.CreateCharacterAssetParams{
			CharacterID:     c.ID,
			EveTypeID:       eveType.ID,
			IsBlueprintCopy: false,
			IsSingleton:     true,
			ItemID:          42,
			LocationFlag:    app.FlagHangar,
			LocationID:      99,
			LocationType:    app.TypeOther,
			Name:            "Alpha",
			Quantity:        7,
		}
		// when
		err := st.CreateCharacterAsset(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterAsset(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, eveType.ID, x.Type.ID)
				assert.Equal(t, eveType.Name, x.Type.Name)
				assert.False(t, x.IsBlueprintCopy)
				assert.True(t, x.IsSingleton)
				assert.Equal(t, int64(42), x.ItemID)
				assert.Equal(t, app.FlagHangar, x.LocationFlag)
				assert.Equal(t, int64(99), x.LocationID)
				assert.Equal(t, app.TypeOther, x.LocationType)
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, int32(7), x.Quantity)
				assert.Equal(t, 1.24, x.Price.ValueOrZero())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		arg := storage.UpdateCharacterAssetParams{
			CharacterID:  c.ID,
			ItemID:       x1.ItemID,
			LocationFlag: app.FlagHangar,
			LocationID:   99,
			LocationType: app.TypeOther,
			Name:         "Alpha",
			Quantity:     7,
		}
		// when
		err := st.UpdateCharacterAsset(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x2, err := st.GetCharacterAsset(ctx, c.ID, x1.ItemID)
			if assert.NoError(t, err) {
				assert.Equal(t, app.FlagHangar, x2.LocationFlag)
				assert.Equal(t, int64(99), x2.LocationID)
				assert.Equal(t, app.TypeOther, x2.LocationType)
				assert.Equal(t, "Alpha", x2.Name)
				assert.Equal(t, int32(7), x2.Quantity)
			}
		}
	})
	t.Run("can list assets in ship hangar", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		location := factory.CreateEveLocationStructure()
		shipCategory := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
		shipGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: shipCategory.ID})
		shipType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: shipGroup.ID})
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
			LocationID:  location.ID,
			EveTypeID:   shipType.ID,
		})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
			LocationID:  location.ID,
		})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
		})
		// when
		oo, err := st.ListCharacterAssetsInShipHangar(ctx, c.ID, location.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 1)
			assert.Equal(t, x1.Type, oo[0].Type)
		}
	})
	t.Run("can delete assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		x2 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		err := st.DeleteCharacterAssets(ctx, c.ID, []int64{x2.ItemID})
		// then
		if assert.NoError(t, err) {
			got, err := st.ListCharacterAssetIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				want := set.Of(x1.ItemID)
				xassert.EqualSet(t, want, got)
			}
		}
	})
	t.Run("can list assets for character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		ca1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		ca2 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		got, err := st.ListCharacterAssets(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := []*app.CharacterAsset{ca1, ca2}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("can list assets for character in item hangar", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		location := factory.CreateEveLocationStructure()
		ca1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID:  c.ID,
			LocationFlag: app.FlagHangar,
			LocationID:   location.ID,
		})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID:  c.ID,
			LocationFlag: app.FlagUnknown,
			LocationID:   location.ID,
		})
		// when
		got, err := st.ListCharacterAssetsInItemHangar(ctx, c.ID, ca1.LocationID)
		// then
		if assert.NoError(t, err) {
			want := []*app.CharacterAsset{ca1}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("can list assets for character in location", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		ca1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		got, err := st.ListCharacterAssetsInLocation(ctx, c.ID, ca1.LocationID)
		// then
		if assert.NoError(t, err) {
			want := []*app.CharacterAsset{ca1}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("can list all assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		ca1 := factory.CreateCharacterAsset()
		ca2 := factory.CreateCharacterAsset()
		// when
		got, err := st.ListAllCharacterAssets(ctx)
		// then
		if assert.NoError(t, err) {
			want := []*app.CharacterAsset{ca1, ca2}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("can calculate total asset value for character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		ca1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
			Quantity:    1,
		})
		ca2 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
			Quantity:    2,
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
		got, err := st.CalculateCharacterAssetTotalValue(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.InDelta(t, 500.5, got, 0.1)
		}
	})
	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetCharacterAsset(ctx, 1, 2)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
