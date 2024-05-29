package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
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
		arg := storage.CreateCharacterAssetParams{
			CharacterID:     c.ID,
			EveTypeID:       eveType.ID,
			IsBlueprintCopy: true,
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
				assert.True(t, x.IsBlueprintCopy)
				assert.True(t, x.IsSingleton)
				assert.Equal(t, int64(42), x.ItemID)
				assert.Equal(t, "Hangar", x.LocationFlag)
				assert.Equal(t, int64(99), x.LocationID)
				assert.Equal(t, "other", x.LocationType)
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, int32(7), x.Quantity)
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
	t.Run("can list assets", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		x2 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		oo, err := r.ListCharacterAssets(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.New[int64]()
			for _, o := range oo {
				got.Add(o.ItemID)
			}
			want := set.NewFromSlice([]int64{x1.ItemID, x2.ItemID})
			assert.Equal(t, want, got)
		}
	})
	t.Run("can delete excluded assets", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		err := r.DeleteExcludedCharacterAssets(ctx, c.ID, []int64{x1.ItemID})
		// then
		if assert.NoError(t, err) {
			ids, err := r.ListCharacterAssetIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				got := set.NewFromSlice(ids)
				want := set.NewFromSlice([]int64{x1.ItemID})
				assert.Equal(t, want, got)
			}
		}
	})
}
