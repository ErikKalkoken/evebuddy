package assetsync_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetsync"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

var (
	logAttr = slog.Int64("ownerID", 42)
	flags   = map[string]app.LocationFlag{"Hangar": app.FlagHangar}
	types   = map[string]app.LocationType{
		"":             app.TypeUndefined,
		"station":      app.TypeStation,
		"solar_system": app.TypeSolarSystem,
		"item":         app.TypeItem,
		"other":        app.TypeOther,
	}
	ship = &app.EveType{
		ID: 3516,
		Group: &app.EveGroup{
			Category: &app.EveCategory{ID: app.EveCategoryShip},
		},
	}
)

func TestExtractIDs(t *testing.T) {
	typeIDs, locationIDs := assetsync.ExtractIDs([]assetsync.ESIAsset{
		{ItemId: 1, LocationId: 100, TypeId: 10},
		{ItemId: 2, LocationId: 1, TypeId: 20}, // inside item 1
		{ItemId: 3, LocationId: 100, TypeId: 10},
	})
	xassert.Equal(t, set.Of[int64](10, 20), typeIDs)
	xassert.Equal(t, set.Of[int64](100), locationIDs)
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	noCreate := func(context.Context, assetsync.Item) error {
		t.Fatal("create should not be called")
		return nil
	}
	noUpdate := func(context.Context, assetsync.Item) error {
		t.Fatal("update should not be called")
		return nil
	}
	noDelete := func(context.Context, set.Set[int64]) error {
		t.Fatal("delete should not be called")
		return nil
	}
	t.Run("creates new assets with parsed values", func(t *testing.T) {
		isCopy := true
		var got []assetsync.Item
		err := assetsync.Sync(
			ctx,
			[]assetsync.ESIAsset{{
				IsBlueprintCopy: &isCopy,
				IsSingleton:     true,
				ItemId:          1,
				LocationFlag:    "Hangar",
				LocationId:      100,
				LocationType:    "station",
				Quantity:        1,
				TypeId:          10,
			}},
			set.Set[int64]{},
			flags,
			types,
			func(_ context.Context, item assetsync.Item) error {
				got = append(got, item)
				return nil
			},
			noUpdate,
			noDelete,
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, []assetsync.Item{{
			IsBlueprintCopy: optional.New(true),
			IsSingleton:     true,
			ItemID:          1,
			LocationFlag:    app.FlagHangar,
			LocationID:      100,
			LocationType:    app.TypeStation,
			Quantity:        1,
			TypeID:          10,
		}}, got)
	})
	t.Run("updates existing assets and deletes obsolete ones", func(t *testing.T) {
		var updated []int64
		var deleted set.Set[int64]
		err := assetsync.Sync(
			ctx,
			[]assetsync.ESIAsset{{ItemId: 1, LocationFlag: "Hangar"}},
			set.Of[int64](1, 2),
			flags,
			types,
			noCreate,
			func(_ context.Context, item assetsync.Item) error {
				updated = append(updated, item.ItemID)
				return nil
			},
			func(_ context.Context, ids set.Set[int64]) error {
				deleted = ids
				return nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, []int64{1}, updated)
		xassert.Equal(t, set.Of[int64](2), deleted)
	})
	t.Run("maps location values", func(t *testing.T) {
		for _, tc := range []struct {
			name         string
			flag         string
			locationType string
			wantFlag     app.LocationFlag
			wantType     app.LocationType
		}{
			{"station", "Hangar", "station", app.FlagHangar, app.TypeStation},
			{"solar system", "Hangar", "solar_system", app.FlagHangar, app.TypeSolarSystem},
			{"item", "Hangar", "item", app.FlagHangar, app.TypeItem},
			{"other", "Hangar", "other", app.FlagHangar, app.TypeOther},
			{"undefined type", "Hangar", "", app.FlagHangar, app.TypeUndefined},
			{"unknown type", "Hangar", "invalid", app.FlagHangar, app.TypeUnknown},
			{"unknown flag", "invalid", "station", app.FlagUnknown, app.TypeStation},
			{"flag of other owner", "CorpSAG1", "station", app.FlagUnknown, app.TypeStation},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var got assetsync.Item
				err := assetsync.Sync(
					ctx,
					[]assetsync.ESIAsset{{ItemId: 1, LocationFlag: tc.flag, LocationType: tc.locationType}},
					set.Set[int64]{},
					flags,
					types,
					func(_ context.Context, item assetsync.Item) error {
						got = item
						return nil
					},
					noUpdate,
					noDelete,
					logAttr,
				)
				require.NoError(t, err)
				xassert.Equal(t, tc.wantFlag, got.LocationFlag)
				xassert.Equal(t, tc.wantType, got.LocationType)
			})
		}
	})
	t.Run("corrects quantity of singletons", func(t *testing.T) {
		for _, tc := range []struct {
			name         string
			existing     bool
			singleton    bool
			quantity     int64
			wantQuantity int64
		}{
			{name: "new singleton", singleton: true, quantity: 5, wantQuantity: 1},
			{name: "existing singleton", existing: true, singleton: true, quantity: 5, wantQuantity: 1},
			{name: "new stack", quantity: 5, wantQuantity: 5},
			{name: "existing stack", existing: true, quantity: 5, wantQuantity: 5},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var currentIDs set.Set[int64]
				if tc.existing {
					currentIDs.Add(1)
				}
				var got assetsync.Item
				store := func(_ context.Context, item assetsync.Item) error {
					got = item
					return nil
				}
				err := assetsync.Sync(
					ctx,
					[]assetsync.ESIAsset{{IsSingleton: tc.singleton, ItemId: 1, Quantity: tc.quantity}},
					currentIDs,
					flags,
					types,
					store,
					store,
					noDelete,
					logAttr,
				)
				require.NoError(t, err)
				xassert.Equal(t, tc.singleton, got.IsSingleton)
				xassert.Equal(t, tc.wantQuantity, got.Quantity)
			})
		}
	})
	t.Run("returns errors from callbacks", func(t *testing.T) {
		wantErr := errors.New("failed")
		fail := func(context.Context, assetsync.Item) error { return wantErr }
		ignore := func(context.Context, assetsync.Item) error { return nil }
		failDelete := func(context.Context, set.Set[int64]) error { return wantErr }
		assets := []assetsync.ESIAsset{{ItemId: 1}}
		err := assetsync.Sync(ctx, assets, set.Set[int64]{}, flags, types, fail, noUpdate, noDelete, logAttr)
		require.ErrorIs(t, err, wantErr)
		err = assetsync.Sync(ctx, assets, set.Of[int64](1), flags, types, noCreate, fail, noDelete, logAttr)
		require.ErrorIs(t, err, wantErr)
		err = assetsync.Sync(ctx, assets, set.Of[int64](1, 2), flags, types, noCreate, ignore, failDelete, logAttr)
		require.ErrorIs(t, err, wantErr)
	})
}

func TestSyncNames(t *testing.T) {
	ctx := context.Background()
	named := func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
		return []assetsync.ESIAssetName{{ItemId: 1, Name: "New Name"}}, nil
	}
	unnamed := func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
		return []assetsync.ESIAssetName{{ItemId: 1, Name: "None"}}, nil
	}
	failed := func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
		return nil, errors.New("failed")
	}
	for _, tc := range []struct {
		name      string
		singleton bool
		stored    string
		fetch     func(context.Context, []int64) ([]assetsync.ESIAssetName, error)
		want      map[int64]string
	}{
		{name: "stores new name", singleton: true, stored: "Old Name", fetch: named, want: map[int64]string{1: "New Name"}},
		{name: "skips unchanged name", singleton: true, stored: "New Name", fetch: named, want: map[int64]string{}},
		{name: "clears name when asset can not have one", stored: "Old Name", fetch: named, want: map[int64]string{1: ""}},
		{name: "clears name when removed in game", singleton: true, stored: "Old Name", fetch: unnamed, want: map[int64]string{1: ""}},
		{name: "keeps name when request fails", singleton: true, stored: "Old Name", fetch: failed, want: map[int64]string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := make(map[int64]string)
			err := assetsync.SyncNames(
				ctx,
				[]*app.Asset{{IsSingleton: tc.singleton, ItemID: 1, Name: tc.stored, Type: ship}},
				tc.fetch,
				func(_ context.Context, itemID int64, name string) error {
					got[itemID] = name
					return nil
				},
				nil,
				logAttr,
			)
			require.NoError(t, err)
			xassert.Equal(t, tc.want, got)
		})
	}
	t.Run("only fetches names for assets which can have one", func(t *testing.T) {
		var fetched []int64
		err := assetsync.SyncNames(
			ctx,
			[]*app.Asset{
				{IsSingleton: true, ItemID: 1, Type: ship},
				{ItemID: 2, Type: ship},
			},
			func(_ context.Context, ids []int64) ([]assetsync.ESIAssetName, error) {
				fetched = append(fetched, ids...)
				return nil, nil
			},
			func(context.Context, int64, string) error { return nil },
			nil,
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, []int64{1}, fetched)
	})
	t.Run("stores adjusted name", func(t *testing.T) {
		got := make(map[int64]string)
		err := assetsync.SyncNames(
			ctx,
			[]*app.Asset{{IsSingleton: true, ItemID: 1, Name: "Old Name", Type: ship}},
			named,
			func(_ context.Context, itemID int64, name string) error {
				got[itemID] = name
				return nil
			},
			func(a *app.Asset, name string) string {
				xassert.Equal(t, 1, a.ItemID)
				return "Adjusted " + name
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, map[int64]string{1: "Adjusted New Name"}, got)
	})
	t.Run("returns error when context is canceled while fetching", func(t *testing.T) {
		var logs bytes.Buffer
		defer slog.SetDefault(slog.Default())
		slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		err := assetsync.SyncNames(
			ctx,
			[]*app.Asset{{IsSingleton: true, ItemID: 1, Name: "Old Name", Type: ship}},
			func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
				cancel()
				return nil, context.Canceled
			},
			func(context.Context, int64, string) error {
				t.Fatal("updateName should not be called")
				return nil
			},
			nil,
			logAttr,
		)
		require.ErrorIs(t, err, context.Canceled)
		assert.NotContains(t, logs.String(), "Failed to fetch asset names")
	})
	t.Run("returns errors from updateName", func(t *testing.T) {
		wantErr := errors.New("failed")
		fail := func(context.Context, int64, string) error { return wantErr }
		// when storing a fetched name
		err := assetsync.SyncNames(ctx, []*app.Asset{{IsSingleton: true, ItemID: 1, Type: ship}}, named, fail, nil, logAttr)
		require.ErrorIs(t, err, wantErr)
		// when clearing a stored name
		err = assetsync.SyncNames(ctx, []*app.Asset{{ItemID: 1, Name: "Old Name", Type: ship}}, named, fail, nil, logAttr)
		require.ErrorIs(t, err, wantErr)
	})
}
