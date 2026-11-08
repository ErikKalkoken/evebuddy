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

func TestCorporationAsset_LocationValidation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		flag    app.LocationFlag
		wantErr bool
	}{
		{name: "accepts flag of owner", flag: app.FlagCorpSAG1},
		{name: "accepts unknown flag", flag: app.FlagUnknown},
		{name: "rejects flag of other owner", flag: app.FlagCorpseBay, wantErr: true},
		{name: "rejects undefined flag", flag: app.FlagUndefined, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			c := factory.CreateCorporation()
			eveType := factory.CreateEveType()
			existing := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
			errUpdate := st.ReplaceCorporationAssets(ctx, c.ID, []storage.ReplaceCorporationAssetParams{{
				ItemID:       existing.ItemID,
				LocationFlag: tc.flag,
				LocationID:   existing.LocationID,
				TypeID:       existing.Type.ID,
			}})
			errCreate := st.CreateCorporationAsset(ctx, storage.CreateCorporationAssetParams{
				CorporationID: c.ID,
				EveTypeID:     eveType.ID,
				ItemID:        42,
				LocationFlag:  tc.flag,
			})
			if tc.wantErr {
				require.ErrorIs(t, errCreate, app.ErrInvalid)
				require.ErrorIs(t, errUpdate, app.ErrInvalid)
				return
			}
			require.NoError(t, errCreate)
			require.NoError(t, errUpdate)
			created, err := st.GetCorporationAsset(ctx, c.ID, 42)
			require.NoError(t, err)
			xassert.Equal(t, tc.flag, created.LocationFlag)
			updated, err := st.GetCorporationAsset(ctx, c.ID, existing.ItemID)
			require.NoError(t, err)
			xassert.Equal(t, tc.flag, updated.LocationFlag)
		})
	}
	t.Run("stores unknown location type", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		eveType := factory.CreateEveType()
		err := st.CreateCorporationAsset(ctx, storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			EveTypeID:     eveType.ID,
			ItemID:        42,
			LocationFlag:  app.FlagHangar,
			LocationType:  app.TypeUnknown,
		})
		require.NoError(t, err)
		x, err := st.GetCorporationAsset(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, app.TypeUnknown, x.LocationType)
	})
}

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
			TypeID:       eveType.ID,
			AveragePrice: optional.New(1.24),
		})
		// when
		err := st.CreateCorporationAsset(ctx, storage.CreateCorporationAssetParams{
			CorporationID:   c.ID,
			EveTypeID:       eveType.ID,
			IsBlueprintCopy: optional.New(false),
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
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			EveTypeID:     blueprintType.ID,
			Quantity:      1,
		}) // blueprint originals must be excluded
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID:   c.ID,
			EveTypeID:       blueprintType.ID,
			IsBlueprintCopy: optional.New(true),
			Quantity:        1,
		}) // blueprint copies must be excluded
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       blueprintType.ID,
			AveragePrice: optional.New(66.6),
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
	t.Run("returns blueprint copy without price", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		ca := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID:   c.ID,
			IsBlueprintCopy: optional.New(true),
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca.Type.ID,
			AveragePrice: optional.New(100.1),
		})
		// when
		o, err := st.GetCorporationAsset(ctx, c.ID, ca.ItemID)
		require.NoError(t, err)
		oo1, err := st.ListCorporationAssets(ctx, c.ID)
		require.NoError(t, err)
		oo2, err := st.ListAllCorporationAssets(ctx)
		require.NoError(t, err)
		// then
		assert.True(t, o.Price.IsEmpty())
		require.Len(t, oo1, 1)
		assert.True(t, oo1[0].Price.IsEmpty())
		require.Len(t, oo2, 1)
		assert.True(t, oo2[0].Price.IsEmpty())
	})
}

func TestStorage_ReplaceCorporationAssets(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should create new assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		eveType := factory.CreateEveType()
		location := factory.CreateEveLocationStructure()
		// when
		err := st.ReplaceCorporationAssets(ctx, c.ID, []storage.ReplaceCorporationAssetParams{
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
		x1, err := st.GetCorporationAsset(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.EqualOptional(t, true, x1.IsBlueprintCopy)
		assert.True(t, x1.IsSingleton)
		xassert.Equal(t, app.FlagHangar, x1.LocationFlag)
		xassert.Equal(t, location.ID, x1.LocationID)
		xassert.Equal(t, app.TypeStation, x1.LocationType)
		xassert.Equal(t, "Alpha", x1.Name)
		xassert.Equal(t, 1, x1.Quantity)
		xassert.Equal(t, eveType.ID, x1.Type.ID)
		x2, err := st.GetCorporationAsset(ctx, c.ID, 43)
		require.NoError(t, err)
		xassert.Equal(t, "", x2.Name)
		xassert.Equal(t, 5, x2.Quantity)
	})
	t.Run("should update existing assets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		x1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		location := factory.CreateEveLocationStructure()
		// when
		err := st.ReplaceCorporationAssets(ctx, c.ID, []storage.ReplaceCorporationAssetParams{{
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
		x2, err := st.GetCorporationAsset(ctx, c.ID, x1.ItemID)
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
				c := factory.CreateCorporation()
				x1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
					CorporationID: c.ID,
					IsSingleton:   true,
					Name:          "Bravo",
				})
				// when
				err := st.ReplaceCorporationAssets(ctx, c.ID, []storage.ReplaceCorporationAssetParams{{
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
				x2, err := st.GetCorporationAsset(ctx, c.ID, x1.ItemID)
				require.NoError(t, err)
				xassert.Equal(t, tc.wantName, x2.Name)
			})
		}
	})
	t.Run("should keep the last copy of duplicate args", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		x1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		// when
		err := st.ReplaceCorporationAssets(ctx, c.ID, []storage.ReplaceCorporationAssetParams{
			{ItemID: x1.ItemID, LocationFlag: x1.LocationFlag, LocationID: x1.LocationID, Quantity: 3, TypeID: x1.Type.ID},
			{ItemID: x1.ItemID, LocationFlag: x1.LocationFlag, LocationID: x1.LocationID, Quantity: 4, TypeID: x1.Type.ID},
			{ItemID: 42, LocationFlag: x1.LocationFlag, LocationID: x1.LocationID, Quantity: 5, TypeID: x1.Type.ID},
			{ItemID: 42, LocationFlag: x1.LocationFlag, LocationID: x1.LocationID, Quantity: 6, TypeID: x1.Type.ID},
		})
		// then
		require.NoError(t, err)
		got1, err := st.GetCorporationAsset(ctx, c.ID, x1.ItemID)
		require.NoError(t, err)
		xassert.Equal(t, 4, got1.Quantity)
		got2, err := st.GetCorporationAsset(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, 6, got2.Quantity)
	})
	t.Run("should delete assets missing from args", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		x1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		// when
		err := st.ReplaceCorporationAssets(ctx, c.ID, []storage.ReplaceCorporationAssetParams{{
			ItemID:       x1.ItemID,
			LocationFlag: x1.LocationFlag,
			LocationID:   x1.LocationID,
			LocationType: x1.LocationType,
			Quantity:     int64(x1.Quantity),
			TypeID:       x1.Type.ID,
		}})
		// then
		require.NoError(t, err)
		got, err := st.ListCorporationAssetIDs(ctx, c.ID)
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
		c := factory.CreateCorporation()
		x1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		// when
		err := st.ReplaceCorporationAssets(ctx, c.ID, []storage.ReplaceCorporationAssetParams{{
			ItemID:       x1.ItemID,
			LocationFlag: x1.LocationFlag,
			LocationID:   x1.LocationID,
			LocationType: x1.LocationType,
			Quantity:     int64(x1.Quantity),
			TypeID:       x1.Type.ID,
		}})
		// then
		require.NoError(t, err)
		got, err := st.ListCorporationAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(x1.ItemID), got)
	})
	t.Run("should delete all assets when args are empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		// when
		err := st.ReplaceCorporationAssets(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		got, err := st.ListCorporationAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, got.Size())
	})
	t.Run("should not touch assets of other corporations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		other := factory.CreateCorporationAsset()
		// when
		err := st.ReplaceCorporationAssets(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		got, err := st.ListCorporationAssetIDs(ctx, other.CorporationID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(other.ItemID), got)
	})
	t.Run("should store nothing when an arg is invalid", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			modify func(arg *storage.ReplaceCorporationAssetParams)
		}{
			{name: "type ID is zero", modify: func(arg *storage.ReplaceCorporationAssetParams) { arg.TypeID = 0 }},
			{name: "item ID is zero", modify: func(arg *storage.ReplaceCorporationAssetParams) { arg.ItemID = 0 }},
			{name: "location flag of other owner", modify: func(arg *storage.ReplaceCorporationAssetParams) { arg.LocationFlag = app.FlagCorpseBay }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCorporation()
				x1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
				x2 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
				invalid := storage.ReplaceCorporationAssetParams{
					ItemID:       42,
					LocationFlag: app.FlagHangar,
					LocationID:   x1.LocationID,
					LocationType: app.TypeOther,
					Quantity:     1,
					TypeID:       x1.Type.ID,
				}
				tc.modify(&invalid)
				// when
				err := st.ReplaceCorporationAssets(ctx, c.ID, []storage.ReplaceCorporationAssetParams{
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
				got, err := st.ListCorporationAssets(ctx, c.ID)
				require.NoError(t, err)
				assert.ElementsMatch(t, []*app.CorporationAsset{x1, x2}, got)
			})
		}
	})
	t.Run("should return error when corporation ID is zero", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.ReplaceCorporationAssets(ctx, 0, nil)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}
