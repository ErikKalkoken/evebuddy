package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestCharacterAsset(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
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
			LocationFlag:    "Hangar",
			LocationID:      99,
			LocationType:    "other",
			Name:            "Alpha",
			Quantity:        7,
		}
		// when
		err := r.CreateCharacterAsset(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterAsset(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, eveType.ID, x.EveType.ID)
				assert.Equal(t, eveType.Name, x.EveType.Name)
				assert.False(t, x.IsBlueprintCopy)
				assert.True(t, x.IsSingleton)
				assert.Equal(t, int64(42), x.ItemID)
				assert.Equal(t, "Hangar", x.LocationFlag)
				assert.Equal(t, int64(99), x.LocationID)
				assert.Equal(t, "other", x.LocationType)
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, int32(7), x.Quantity)
				assert.Equal(t, 1.24, x.Price.ValueOrZero())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		arg := storage.UpdateCharacterAssetParams{
			CharacterID:  c.ID,
			ItemID:       x1.ItemID,
			LocationFlag: "Hangar",
			LocationID:   99,
			LocationType: "other",
			Name:         "Alpha",
			Quantity:     7,
		}
		// when
		err := r.UpdateCharacterAsset(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x2, err := r.GetCharacterAsset(ctx, c.ID, x1.ItemID)
			if assert.NoError(t, err) {
				assert.Equal(t, "Hangar", x2.LocationFlag)
				assert.Equal(t, int64(99), x2.LocationID)
				assert.Equal(t, "other", x2.LocationType)
				assert.Equal(t, "Alpha", x2.Name)
				assert.Equal(t, int32(7), x2.Quantity)
			}
		}
	})
	t.Run("can list assets in ship hangar", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		location := factory.CreateLocationStructure()
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
		oo, err := r.ListCharacterAssetsInShipHangar(ctx, c.ID, location.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 1)
			assert.Equal(t, x1.EveType, oo[0].EveType)
		}
	})
	t.Run("can delete assets", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		x2 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		err := r.DeleteCharacterAssets(ctx, c.ID, []int64{x2.ItemID})
		// then
		if assert.NoError(t, err) {
			ids, err := r.ListCharacterAssetIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, []int64{x1.ItemID}, ids)
			}
		}
	})
	t.Run("can list assets for character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		ca1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		ca2 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		got, err := r.ListCharacterAssets(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := []*app.CharacterAsset{ca1, ca2}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("can list assets for character in item hangar", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		location := factory.CreateLocationStructure()
		ca1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID:  c.ID,
			LocationFlag: "Hangar",
			LocationID:   location.ID,
		})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID:  c.ID,
			LocationFlag: "XXX",
			LocationID:   location.ID,
		})
		// when
		got, err := r.ListCharacterAssetsInItemHangar(ctx, c.ID, ca1.LocationID)
		// then
		if assert.NoError(t, err) {
			want := []*app.CharacterAsset{ca1}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("can list assets for character in location", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		ca1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		got, err := r.ListCharacterAssetsInLocation(ctx, c.ID, ca1.LocationID)
		// then
		if assert.NoError(t, err) {
			want := []*app.CharacterAsset{ca1}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("can list all assets", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		ca1 := factory.CreateCharacterAsset()
		ca2 := factory.CreateCharacterAsset()
		// when
		got, err := r.ListAllCharacterAssets(ctx)
		// then
		if assert.NoError(t, err) {
			want := []*app.CharacterAsset{ca1, ca2}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("can calculate total asset value for character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		ca1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
			Quantity:    1,
		})
		ca2 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
			Quantity:    2,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca1.EveType.ID,
			AveragePrice: 100.1,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca2.EveType.ID,
			AveragePrice: 200.2,
		})
		// when
		got, err := r.CalculateCharacterAssetTotalValue(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.InDelta(t, 500.5, got, 0.1)
		}
	})

}
