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
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterAsset_LocationValidation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		flag    app.LocationFlag
		wantErr bool
	}{
		{name: "accepts flag of owner", flag: app.FlagCorpseBay},
		{name: "accepts unknown flag", flag: app.FlagUnknown},
		{name: "rejects flag of other owner", flag: app.FlagCorpSAG1, wantErr: true},
		{name: "rejects undefined flag", flag: app.FlagUndefined, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			c := factory.CreateCharacterFull()
			eveType := factory.CreateEveType()
			existing := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
			errUpdate := st.ReplaceCharacterAssets(ctx, c.ID, []storage.ReplaceCharacterAssetParams{{
				ItemID:       existing.ItemID,
				LocationFlag: tc.flag,
				LocationID:   existing.LocationID,
				TypeID:       existing.Type.ID,
			}})
			errCreate := st.CreateCharacterAsset(ctx, storage.CreateCharacterAssetParams{
				CharacterID:  c.ID,
				ItemID:       42,
				LocationFlag: tc.flag,
				TypeID:       eveType.ID,
			})
			if tc.wantErr {
				require.ErrorIs(t, errCreate, app.ErrInvalid)
				require.ErrorIs(t, errUpdate, app.ErrInvalid)
				return
			}
			require.NoError(t, errCreate)
			require.NoError(t, errUpdate)
			created, err := st.GetCharacterAsset(ctx, c.ID, 42)
			require.NoError(t, err)
			xassert.Equal(t, tc.flag, created.LocationFlag)
			updated, err := st.GetCharacterAsset(ctx, c.ID, existing.ItemID)
			require.NoError(t, err)
			xassert.Equal(t, tc.flag, updated.LocationFlag)
		})
	}
	t.Run("stores unknown location type", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		eveType := factory.CreateEveType()
		err := st.CreateCharacterAsset(ctx, storage.CreateCharacterAssetParams{
			CharacterID:  c.ID,
			ItemID:       42,
			LocationFlag: app.FlagHangar,
			LocationType: app.TypeUnknown,
			TypeID:       eveType.ID,
		})
		require.NoError(t, err)
		x, err := st.GetCharacterAsset(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, app.TypeUnknown, x.LocationType)
	})
}

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
			TypeID:       eveType.ID,
			AveragePrice: optional.New(1.24),
		})
		// when
		err := st.CreateCharacterAsset(ctx, storage.CreateCharacterAssetParams{
			CharacterID:  c.ID,
			TypeID:       eveType.ID,
			IsSingleton:  true,
			ItemID:       42,
			LocationFlag: app.FlagHangar,
			LocationID:   99,
			LocationType: app.TypeOther,
			Name:         "Alpha",
			Quantity:     7,
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterAsset(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, eveType.ID, x.Type.ID)
		xassert.Equal(t, eveType.Name, x.Type.Name)
		assert.False(t, x.IsBlueprintCopy.ValueOrZero())
		assert.True(t, x.IsSingleton)
		xassert.Equal(t, 42, x.ItemID)
		xassert.Equal(t, app.FlagHangar, x.LocationFlag)
		xassert.Equal(t, 99, x.LocationID)
		xassert.Equal(t, app.TypeOther, x.LocationType)
		xassert.Equal(t, "Alpha", x.Name)
		xassert.Equal(t, 7, x.Quantity)
		xassert.EqualOptional(t, 1.24, x.Price)
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
		require.NoError(t, err)
		want := []*app.CharacterAsset{ca1, ca2}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can list all assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		ca1 := factory.CreateCharacterAsset()
		ca2 := factory.CreateCharacterAsset()
		// when
		got, err := st.ListAllCharacterAssets(ctx)
		// then
		require.NoError(t, err)
		want := []*app.CharacterAsset{ca1, ca2}
		assert.ElementsMatch(t, want, got)
	})

	t.Run("can calculate total asset value for character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
			TypeID:       ca1.Type.ID,
			AveragePrice: optional.New(100.1),
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca2.Type.ID,
			AveragePrice: optional.New(200.2),
		})
		blueprintCategory := factory.CreateEveCategory(storage.CreateEveCategoryParams{
			ID:   app.EveCategoryBlueprint,
			Name: "Blueprint",
		})
		blueprintGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: blueprintCategory.ID,
		})
		blueprintType := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID: blueprintGroup.ID,
		})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
			Quantity:    1,
			TypeID:      blueprintType.ID,
		}) // blueprints must be excluded
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       blueprintType.ID,
			AveragePrice: optional.New(66.6),
		})

		// when
		got, err := st.CalculateCharacterAssetTotalValue(ctx, c.ID)

		// then
		require.NoError(t, err)
		assert.InDelta(t, 500.5, got, 0.1)
	})

	t.Run("should calculate total asset value from own assets only", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		ca1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
			Quantity:    1,
		})
		ca2 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			Quantity: 1,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca1.Type.ID,
			AveragePrice: optional.New(100.1),
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca2.Type.ID,
			AveragePrice: optional.New(200.2),
		})
		// when
		got, err := st.CalculateCharacterAssetTotalValue(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.InDelta(t, 100.1, got, 0.1)
	})

	t.Run("should return zero total asset value when character has no assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		got, err := st.CalculateCharacterAssetTotalValue(ctx, c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0.0, got)
	})

	t.Run("should return blueprint copy without price", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		ca := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID:     c.ID,
			IsBlueprintCopy: optional.New(true),
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca.Type.ID,
			AveragePrice: optional.New(100.1),
		})
		// when
		x, err := st.GetCharacterAsset(ctx, c.ID, ca.ItemID)
		// then
		require.NoError(t, err)
		assert.True(t, x.Price.IsEmpty())
	})

	t.Run("should list blueprint copy without price", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		ca := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID:     c.ID,
			IsBlueprintCopy: optional.New(true),
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca.Type.ID,
			AveragePrice: optional.New(100.1),
		})
		// when
		oo1, err := st.ListCharacterAssets(ctx, c.ID)
		require.NoError(t, err)
		oo2, err := st.ListAllCharacterAssets(ctx)
		require.NoError(t, err)
		// then
		require.Len(t, oo1, 1)
		assert.True(t, oo1[0].Price.IsEmpty())
		require.Len(t, oo2, 1)
		assert.True(t, oo2[0].Price.IsEmpty())
	})

	t.Run("should map unknown location flag to FlagUnknown", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		ca := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
		})
		_, err := db.Exec("UPDATE character_assets SET location_flag = 'NotAFlag' WHERE item_id = ?", ca.ItemID)
		require.NoError(t, err)
		// when
		x, err := st.GetCharacterAsset(ctx, c.ID, ca.ItemID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, app.FlagUnknown, x.LocationFlag)
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

func TestStorage_ReplaceCharacterAssets(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should create new assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		eveType := factory.CreateEveType()
		location := factory.CreateEveLocationStructure()
		// when
		err := st.ReplaceCharacterAssets(ctx, c.ID, []storage.ReplaceCharacterAssetParams{
			{
				IsBlueprintCopy: optional.New(true),
				IsSingleton:     true,
				ItemID:          42,
				LocationFlag:    app.FlagHangar,
				LocationID:      location.ID,
				LocationType:    app.TypeStation,
				Name:            optional.New("Alpha"),
				Quantity:        1,
				TypeID:          eveType.ID,
			},
			{
				ItemID:       43,
				LocationFlag: app.FlagCargo,
				LocationID:   location.ID,
				LocationType: app.TypeOther,
				Quantity:     5,
				TypeID:       eveType.ID,
			},
		})
		// then
		require.NoError(t, err)
		x1, err := st.GetCharacterAsset(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.EqualOptional(t, true, x1.IsBlueprintCopy)
		assert.True(t, x1.IsSingleton)
		xassert.Equal(t, app.FlagHangar, x1.LocationFlag)
		xassert.Equal(t, location.ID, x1.LocationID)
		xassert.Equal(t, app.TypeStation, x1.LocationType)
		xassert.Equal(t, "Alpha", x1.Name)
		xassert.Equal(t, 1, x1.Quantity)
		xassert.Equal(t, eveType.ID, x1.Type.ID)
		x2, err := st.GetCharacterAsset(ctx, c.ID, 43)
		require.NoError(t, err)
		xassert.Equal(t, "", x2.Name)
		xassert.Equal(t, 5, x2.Quantity)
	})
	t.Run("should update existing assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		location := factory.CreateEveLocationStructure()
		// when
		err := st.ReplaceCharacterAssets(ctx, c.ID, []storage.ReplaceCharacterAssetParams{{
			IsSingleton:  true,
			ItemID:       x1.ItemID,
			LocationFlag: app.FlagCargo,
			LocationID:   location.ID,
			LocationType: app.TypeStation,
			Quantity:     1,
			TypeID:       x1.Type.ID,
		}})
		// then
		require.NoError(t, err)
		x2, err := st.GetCharacterAsset(ctx, c.ID, x1.ItemID)
		require.NoError(t, err)
		assert.True(t, x2.IsSingleton)
		xassert.Equal(t, app.FlagCargo, x2.LocationFlag)
		xassert.Equal(t, location.ID, x2.LocationID)
		xassert.Equal(t, app.TypeStation, x2.LocationType)
		xassert.Equal(t, 1, x2.Quantity)
	})
	t.Run("should set, clear or keep names of existing assets", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			argName  optional.Optional[string]
			wantName string
		}{
			{name: "set", argName: optional.New("Alpha"), wantName: "Alpha"},
			{name: "clear", argName: optional.New(""), wantName: ""},
			{name: "keep", argName: optional.Optional[string]{}, wantName: "Bravo"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCharacterFull()
				x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
					CharacterID: c.ID,
					IsSingleton: true,
					Name:        "Bravo",
				})
				// when
				err := st.ReplaceCharacterAssets(ctx, c.ID, []storage.ReplaceCharacterAssetParams{{
					IsSingleton:  true,
					ItemID:       x1.ItemID,
					LocationFlag: x1.LocationFlag,
					LocationID:   x1.LocationID,
					LocationType: x1.LocationType,
					Name:         tc.argName,
					Quantity:     1,
					TypeID:       x1.Type.ID,
				}})
				// then
				require.NoError(t, err)
				x2, err := st.GetCharacterAsset(ctx, c.ID, x1.ItemID)
				require.NoError(t, err)
				xassert.Equal(t, tc.wantName, x2.Name)
			})
		}
	})
	t.Run("should keep the last copy of duplicate args", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		err := st.ReplaceCharacterAssets(ctx, c.ID, []storage.ReplaceCharacterAssetParams{
			{ItemID: x1.ItemID, LocationFlag: x1.LocationFlag, LocationID: x1.LocationID, Quantity: 3, TypeID: x1.Type.ID},
			{ItemID: x1.ItemID, LocationFlag: x1.LocationFlag, LocationID: x1.LocationID, Quantity: 4, TypeID: x1.Type.ID},
			{ItemID: 42, LocationFlag: x1.LocationFlag, LocationID: x1.LocationID, Quantity: 5, TypeID: x1.Type.ID},
			{ItemID: 42, LocationFlag: x1.LocationFlag, LocationID: x1.LocationID, Quantity: 6, TypeID: x1.Type.ID},
		})
		// then
		require.NoError(t, err)
		got1, err := st.GetCharacterAsset(ctx, c.ID, x1.ItemID)
		require.NoError(t, err)
		xassert.Equal(t, 4, got1.Quantity)
		got2, err := st.GetCharacterAsset(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, 6, got2.Quantity)
	})
	t.Run("should delete assets missing from args", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		err := st.ReplaceCharacterAssets(ctx, c.ID, []storage.ReplaceCharacterAssetParams{{
			ItemID:       x1.ItemID,
			LocationFlag: x1.LocationFlag,
			LocationID:   x1.LocationID,
			LocationType: x1.LocationType,
			Quantity:     int64(x1.Quantity),
			TypeID:       x1.Type.ID,
		}})
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(x1.ItemID), got)
	})
	t.Run("should delete assets missing from args with chunking", func(t *testing.T) {
		// given
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		err := st.ReplaceCharacterAssets(ctx, c.ID, []storage.ReplaceCharacterAssetParams{{
			ItemID:       x1.ItemID,
			LocationFlag: x1.LocationFlag,
			LocationID:   x1.LocationID,
			LocationType: x1.LocationType,
			Quantity:     int64(x1.Quantity),
			TypeID:       x1.Type.ID,
		}})
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(x1.ItemID), got)
	})
	t.Run("should delete all assets when args are empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		// when
		err := st.ReplaceCharacterAssets(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, got.Size())
	})
	t.Run("should not touch assets of other characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		other := factory.CreateCharacterAsset()
		// when
		err := st.ReplaceCharacterAssets(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterAssetIDs(ctx, other.CharacterID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(other.ItemID), got)
	})
	t.Run("should store nothing when an arg is invalid", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			modify func(arg *storage.ReplaceCharacterAssetParams)
		}{
			{name: "type ID is zero", modify: func(arg *storage.ReplaceCharacterAssetParams) { arg.TypeID = 0 }},
			{name: "item ID is zero", modify: func(arg *storage.ReplaceCharacterAssetParams) { arg.ItemID = 0 }},
			{name: "location flag of other owner", modify: func(arg *storage.ReplaceCharacterAssetParams) { arg.LocationFlag = app.FlagCorpSAG1 }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCharacterFull()
				x1 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
				x2 := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
				invalid := storage.ReplaceCharacterAssetParams{
					ItemID:       42,
					LocationFlag: app.FlagHangar,
					LocationID:   x1.LocationID,
					LocationType: app.TypeOther,
					Quantity:     1,
					TypeID:       x1.Type.ID,
				}
				tc.modify(&invalid)
				// when
				err := st.ReplaceCharacterAssets(ctx, c.ID, []storage.ReplaceCharacterAssetParams{
					{
						ItemID:       x1.ItemID,
						LocationFlag: x1.LocationFlag,
						LocationID:   x1.LocationID,
						LocationType: x1.LocationType,
						Quantity:     int64(x1.Quantity) + 1,
						TypeID:       x1.Type.ID,
					},
					invalid,
				})
				// then
				assert.ErrorIs(t, err, app.ErrInvalid)
				got, err := st.ListCharacterAssets(ctx, c.ID)
				require.NoError(t, err)
				assert.ElementsMatch(t, []*app.CharacterAsset{x1, x2}, got)
			})
		}
	})
	t.Run("should return error when character ID is zero", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.ReplaceCharacterAssets(ctx, 0, nil)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}
