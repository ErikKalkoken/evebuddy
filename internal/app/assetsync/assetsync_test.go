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
	t.Run("replaces assets with parsed values", func(t *testing.T) {
		// given
		isCopy := true
		var got []assetsync.Item
		// when
		err := assetsync.Sync(
			ctx,
			[]assetsync.ESIAsset{
				{
					IsBlueprintCopy: &isCopy,
					IsSingleton:     true,
					ItemId:          1,
					LocationFlag:    "Hangar",
					LocationId:      100,
					LocationType:    "station",
					Quantity:        1,
					TypeId:          10,
				},
				{
					ItemId:       2,
					LocationFlag: "Hangar",
					LocationId:   1,
					LocationType: "item",
					Quantity:     5,
					TypeId:       20,
				},
			},
			map[int64]string{1: "Alpha"},
			flags,
			types,
			func(_ context.Context, items []assetsync.Item) error {
				got = items
				return nil
			},
			logAttr,
		)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []assetsync.Item{
			{
				IsBlueprintCopy: optional.New(true),
				IsSingleton:     true,
				ItemID:          1,
				LocationFlag:    app.FlagHangar,
				LocationID:      100,
				LocationType:    app.TypeStation,
				Name:            optional.New("Alpha"),
				Quantity:        1,
				TypeID:          10,
			},
			{
				ItemID:       2,
				LocationFlag: app.FlagHangar,
				LocationID:   1,
				LocationType: app.TypeItem,
				Quantity:     5,
				TypeID:       20,
			},
		}, got)
	})
	t.Run("sets or keeps names", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			names map[int64]string
			want  optional.Optional[string]
		}{
			{name: "set", names: map[int64]string{1: "Alpha"}, want: optional.New("Alpha")},
			{name: "clear", names: map[int64]string{1: ""}, want: optional.New("")},
			{name: "keep", names: map[int64]string{}, want: optional.Optional[string]{}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				// given
				var got assetsync.Item
				// when
				err := assetsync.Sync(
					ctx,
					[]assetsync.ESIAsset{{ItemId: 1, LocationFlag: "Hangar"}},
					tc.names,
					flags,
					types,
					func(_ context.Context, items []assetsync.Item) error {
						got = items[0]
						return nil
					},
					logAttr,
				)
				// then
				require.NoError(t, err)
				xassert.Equal(t, tc.want, got.Name)
			})
		}
	})
	t.Run("keeps the last copy of duplicate assets", func(t *testing.T) {
		// given
		var got []assetsync.Item
		// when
		err := assetsync.Sync(
			ctx,
			[]assetsync.ESIAsset{
				{ItemId: 1, LocationFlag: "Hangar", Quantity: 3},
				{ItemId: 2, LocationFlag: "Hangar", Quantity: 4},
				{ItemId: 1, LocationFlag: "Hangar", Quantity: 5},
			},
			nil,
			flags,
			types,
			func(_ context.Context, items []assetsync.Item) error {
				got = items
				return nil
			},
			logAttr,
		)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []assetsync.Item{
			{ItemID: 1, LocationFlag: app.FlagHangar, Quantity: 5},
			{ItemID: 2, LocationFlag: app.FlagHangar, Quantity: 4},
		}, got)
	})
	t.Run("replaces with no items when ESI has no assets", func(t *testing.T) {
		// given
		var called bool
		// when
		err := assetsync.Sync(
			ctx,
			nil,
			nil,
			flags,
			types,
			func(_ context.Context, items []assetsync.Item) error {
				called = true
				xassert.Equal(t, 0, len(items))
				return nil
			},
			logAttr,
		)
		// then
		require.NoError(t, err)
		assert.True(t, called)
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
				// given
				var got assetsync.Item
				// when
				err := assetsync.Sync(
					ctx,
					[]assetsync.ESIAsset{{ItemId: 1, LocationFlag: tc.flag, LocationType: tc.locationType}},
					nil,
					flags,
					types,
					func(_ context.Context, items []assetsync.Item) error {
						got = items[0]
						return nil
					},
					logAttr,
				)
				// then
				require.NoError(t, err)
				xassert.Equal(t, tc.wantFlag, got.LocationFlag)
				xassert.Equal(t, tc.wantType, got.LocationType)
			})
		}
	})
	t.Run("corrects quantity of singletons", func(t *testing.T) {
		for _, tc := range []struct {
			name         string
			singleton    bool
			quantity     int64
			wantQuantity int64
		}{
			{name: "singleton", singleton: true, quantity: 5, wantQuantity: 1},
			{name: "stack", quantity: 5, wantQuantity: 5},
		} {
			t.Run(tc.name, func(t *testing.T) {
				// given
				var got assetsync.Item
				// when
				err := assetsync.Sync(
					ctx,
					[]assetsync.ESIAsset{{IsSingleton: tc.singleton, ItemId: 1, Quantity: tc.quantity}},
					nil,
					flags,
					types,
					func(_ context.Context, items []assetsync.Item) error {
						got = items[0]
						return nil
					},
					logAttr,
				)
				// then
				require.NoError(t, err)
				xassert.Equal(t, tc.singleton, got.IsSingleton)
				xassert.Equal(t, tc.wantQuantity, got.Quantity)
			})
		}
	})
	t.Run("returns error from replace", func(t *testing.T) {
		// given
		wantErr := errors.New("failed")
		// when
		err := assetsync.Sync(
			ctx,
			[]assetsync.ESIAsset{{ItemId: 1}},
			nil,
			flags,
			types,
			func(context.Context, []assetsync.Item) error { return wantErr },
			logAttr,
		)
		// then
		require.ErrorIs(t, err, wantErr)
	})
}

func TestResolveNames(t *testing.T) {
	ctx := context.Background()
	eveTypes := []*app.EveType{ship}
	named := func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
		return []assetsync.ESIAssetName{{ItemId: 1, Name: "New Name"}}, nil
	}
	unnamed := func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
		return []assetsync.ESIAssetName{{ItemId: 1, Name: "None"}}, nil
	}
	failed := func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
		return nil, errors.New("failed")
	}
	unrequested := func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
		return []assetsync.ESIAssetName{{ItemId: 2, Name: "Other Name"}}, nil
	}
	for _, tc := range []struct {
		name      string
		singleton bool
		typeID    int64
		fetch     func(context.Context, []int64) ([]assetsync.ESIAssetName, error)
		want      map[int64]string
	}{
		{name: "returns fetched name", singleton: true, typeID: ship.ID, fetch: named, want: map[int64]string{1: "New Name"}},
		{name: "clears name when asset can not have one", typeID: ship.ID, fetch: named, want: map[int64]string{1: ""}},
		{name: "clears name when removed in game", singleton: true, typeID: ship.ID, fetch: unnamed, want: map[int64]string{1: ""}},
		{name: "keeps name when request fails", singleton: true, typeID: ship.ID, fetch: failed, want: map[int64]string{}},
		{name: "keeps name when type is unknown", singleton: true, typeID: 99, fetch: named, want: map[int64]string{}},
		{name: "ignores names of assets not requested", singleton: true, typeID: ship.ID, fetch: unrequested, want: map[int64]string{1: ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := assetsync.ResolveNames(
				ctx,
				[]assetsync.ESIAsset{{IsSingleton: tc.singleton, ItemId: 1, TypeId: tc.typeID}},
				eveTypes,
				tc.fetch,
				nil,
				logAttr,
			)
			// then
			require.NoError(t, err)
			xassert.Equal(t, tc.want, got)
		})
	}
	t.Run("only fetches names for assets which can have one", func(t *testing.T) {
		// given
		var fetched []int64
		// when
		_, err := assetsync.ResolveNames(
			ctx,
			[]assetsync.ESIAsset{
				{IsSingleton: true, ItemId: 1, TypeId: ship.ID},
				{ItemId: 2, TypeId: ship.ID},
			},
			eveTypes,
			func(_ context.Context, ids []int64) ([]assetsync.ESIAssetName, error) {
				fetched = append(fetched, ids...)
				return nil, nil
			},
			nil,
			logAttr,
		)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []int64{1}, fetched)
	})
	t.Run("returns adjusted name", func(t *testing.T) {
		// when
		got, err := assetsync.ResolveNames(
			ctx,
			[]assetsync.ESIAsset{{IsSingleton: true, ItemId: 1, TypeId: ship.ID}},
			eveTypes,
			named,
			func(typeID int64, name string) string {
				xassert.Equal(t, ship.ID, typeID)
				return "Adjusted " + name
			},
			logAttr,
		)
		// then
		require.NoError(t, err)
		xassert.Equal(t, map[int64]string{1: "Adjusted New Name"}, got)
	})
	t.Run("fetches names in batches", func(t *testing.T) {
		// given
		assets := make([]assetsync.ESIAsset, 1000) // needs two requests
		for i := range assets {
			assets[i] = assetsync.ESIAsset{IsSingleton: true, ItemId: int64(i + 1), TypeId: ship.ID}
		}
		var requests int
		// when
		got, err := assetsync.ResolveNames(
			ctx,
			assets,
			eveTypes,
			func(_ context.Context, ids []int64) ([]assetsync.ESIAssetName, error) {
				requests++
				assert.LessOrEqual(t, len(ids), 999)
				return []assetsync.ESIAssetName{{ItemId: 1, Name: "Alpha"}}, nil
			},
			nil,
			logAttr,
		)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 2, requests)
		assert.Len(t, got, 1000)
		xassert.Equal(t, "Alpha", got[1])
		xassert.Equal(t, "", got[1000])
	})
	t.Run("stops at the first failed request", func(t *testing.T) {
		// given
		assets := make([]assetsync.ESIAsset, 1000) // needs two requests
		for i := range assets {
			assets[i] = assetsync.ESIAsset{IsSingleton: true, ItemId: int64(i + 1), TypeId: ship.ID}
		}
		var requests int
		// when
		got, err := assetsync.ResolveNames(
			ctx,
			assets,
			eveTypes,
			func(_ context.Context, _ []int64) ([]assetsync.ESIAssetName, error) {
				requests++
				return nil, errors.New("failed")
			},
			nil,
			logAttr,
		)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 1, requests)
		assert.Empty(t, got)
	})
	t.Run("keeps names fetched before a failed request", func(t *testing.T) {
		// given
		assets := make([]assetsync.ESIAsset, 1000) // needs two requests
		for i := range assets {
			assets[i] = assetsync.ESIAsset{IsSingleton: true, ItemId: int64(i + 1), TypeId: ship.ID}
		}
		var requests int
		// when
		got, err := assetsync.ResolveNames(
			ctx,
			assets,
			eveTypes,
			func(_ context.Context, _ []int64) ([]assetsync.ESIAssetName, error) {
				requests++
				if requests == 2 {
					return nil, errors.New("failed")
				}
				return []assetsync.ESIAssetName{{ItemId: 1, Name: "Alpha"}}, nil
			},
			nil,
			logAttr,
		)
		// then
		require.NoError(t, err)
		assert.Len(t, got, 999)
		xassert.Equal(t, "Alpha", got[1])
		assert.NotContains(t, got, int64(1000))
	})
	t.Run("makes no requests when no asset can have a name", func(t *testing.T) {
		// when
		got, err := assetsync.ResolveNames(
			ctx,
			[]assetsync.ESIAsset{{ItemId: 1, TypeId: ship.ID}},
			eveTypes,
			func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
				t.Fatal("fetch should not be called")
				return nil, nil
			},
			nil,
			logAttr,
		)
		// then
		require.NoError(t, err)
		xassert.Equal(t, map[int64]string{1: ""}, got)
	})
	t.Run("returns error when context is canceled while fetching", func(t *testing.T) {
		// given
		var logs bytes.Buffer
		defer slog.SetDefault(slog.Default())
		slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		assets := make([]assetsync.ESIAsset, 1000) // needs two requests
		for i := range assets {
			assets[i] = assetsync.ESIAsset{IsSingleton: true, ItemId: int64(i + 1), TypeId: ship.ID}
		}
		var requests int
		// when
		_, err := assetsync.ResolveNames(
			ctx,
			assets,
			eveTypes,
			func(context.Context, []int64) ([]assetsync.ESIAssetName, error) {
				requests++
				cancel()
				return nil, context.Canceled
			},
			nil,
			logAttr,
		)
		// then
		require.ErrorIs(t, err, context.Canceled)
		xassert.Equal(t, 1, requests)
		assert.NotContains(t, logs.String(), "Failed to fetch asset names")
	})
}
