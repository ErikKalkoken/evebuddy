package characterservice

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateCharacterAssetsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("should create new assets from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{
			ID:   app.EveCategoryShip,
			Name: "Ship",
		})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: category.ID,
		})
		ship := factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516, GroupID: group.ID})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets?page=1", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_blueprint_copy": true,
				"is_singleton":      true,
				"item_id":           1000000016835,
				"location_flag":     "Hangar",
				"location_id":       60002959,
				"location_type":     "station",
				"quantity":          1,
				"type_id":           3516,
			}, {
				"is_blueprint_copy": true,
				"is_singleton":      false,
				"item_id":           1000000016836,
				"location_flag":     "Hangar",
				"location_id":       60002959,
				"location_type":     "station",
				"quantity":          1,
				"type_id":           3516,
			}}).HeaderSet(http.Header{"X-Pages": []string{"1"}}),
		)
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets/names", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"item_id": 1000000016835,
					"name":    "Awesome Name",
				},
				{
					"item_id": 1000000016836,
					"name":    "None",
				},
			}),
		)
		// when
		changed, err := s.updateAssetsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterAssets,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 2, ids.Size())
		x, err := st.GetCharacterAsset(ctx, c.ID, 1000000016835)
		require.NoError(t, err)
		xassert.Equal(t, ship.ID, x.Type.ID)
		xassert.Equal(t, ship.Name, x.Type.Name)
		assert.True(t, x.IsBlueprintCopy.ValueOrZero())
		assert.True(t, x.IsSingleton)
		xassert.Equal(t, app.FlagHangar, x.LocationFlag)
		xassert.Equal(t, location.ID, x.LocationID)
		xassert.Equal(t, app.TypeStation, x.LocationType)
		xassert.Equal(t, "Awesome Name", x.Name)
		xassert.Equal(t, 1, x.Quantity)
		x, err = st.GetCharacterAsset(ctx, c.ID, 1000000016836)
		require.NoError(t, err)
		xassert.Equal(t, "", x.Name)
	})
	t.Run("should remove obsolete items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID, ItemID: 1000000019999,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets?page=1", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_blueprint_copy": true,
				"is_singleton":      true,
				"item_id":           1000000016835,
				"location_flag":     "Hangar",
				"location_id":       60002959,
				"location_type":     "station",
				"quantity":          1,
				"type_id":           3516,
			}, {
				"is_blueprint_copy": true,
				"is_singleton":      false,
				"item_id":           1000000016836,
				"location_flag":     "Hangar",
				"location_id":       60002959,
				"location_type":     "station",
				"quantity":          1,
				"type_id":           3516,
			}}).HeaderSet(http.Header{"X-Pages": []string{"1"}}),
		)
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets/names", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"item_id": 1000000016835,
					"name":    "Awesome Name",
				},
				{
					"item_id": 1000000016836,
					"name":    "None",
				},
			}),
		)
		// when
		changed, err := s.updateAssetsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterAssets,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1000000016835, 1000000016836), ids)
	})
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{
			ID:   app.EveCategoryShip,
			Name: "Ship",
		})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: category.ID,
		})
		ship := factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516, GroupID: group.ID})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		pages := "2"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets?page=1", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"is_blueprint_copy": true,
					"is_singleton":      true,
					"item_id":           1000000016835,
					"location_flag":     "Hangar",
					"location_id":       60002959,
					"location_type":     "station",
					"quantity":          1,
					"type_id":           3516,
				},
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets?page=2", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"is_blueprint_copy": true,
					"is_singleton":      false,
					"item_id":           1000000016836,
					"location_flag":     "Hangar",
					"location_id":       60002959,
					"location_type":     "station",
					"quantity":          1,
					"type_id":           3516,
				},
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets/names", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"item_id": 1000000016835,
					"name":    "Awesome Name",
				},
				{
					"item_id": 1000000016836,
					"name":    "None",
				},
			}),
		)
		// when
		changed, err := s.updateAssetsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterAssets,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 2, ids.Size())
		x, err := st.GetCharacterAsset(ctx, c.ID, 1000000016835)
		require.NoError(t, err)
		xassert.Equal(t, ship.ID, x.Type.ID)
		xassert.Equal(t, ship.Name, x.Type.Name)
		assert.True(t, x.IsBlueprintCopy.ValueOrZero())
		assert.True(t, x.IsSingleton)
		xassert.Equal(t, app.FlagHangar, x.LocationFlag)
		xassert.Equal(t, location.ID, x.LocationID)
		xassert.Equal(t, app.TypeStation, x.LocationType)
		xassert.Equal(t, "Awesome Name", x.Name)
		xassert.Equal(t, 1, x.Quantity)
		x, err = st.GetCharacterAsset(ctx, c.ID, 1000000016836)
		require.NoError(t, err)
		xassert.Equal(t, "", x.Name)
	})
}

func TestUpdateCharacterAssetsESI_Singleton(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	const itemID = 1000000016835
	for _, tc := range []struct {
		name              string
		existing          bool
		existingSingleton bool
		existingQuantity  int64
		singleton         bool
		quantity          int64
		wantQuantity      int
	}{
		{name: "corrects quantity of new singleton", singleton: true, quantity: 5, wantQuantity: 1},
		{name: "corrects quantity of existing singleton", existing: true, existingSingleton: true, existingQuantity: 1, singleton: true, quantity: 5, wantQuantity: 1},
		{name: "keeps quantity of new stack", quantity: 5, wantQuantity: 5},
		{name: "updates stack to singleton", existing: true, existingQuantity: 5, singleton: true, quantity: 1, wantQuantity: 1},
		{name: "updates singleton to stack", existing: true, existingSingleton: true, existingQuantity: 1, quantity: 5, wantQuantity: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			c := factory.CreateCharacterFull()
			factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
			et := factory.CreateEveType()
			factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
			if tc.existing {
				factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
					CharacterID: c.ID,
					IsSingleton: tc.existingSingleton,
					ItemID:      itemID,
					Quantity:    tc.existingQuantity,
					TypeID:      et.ID,
				})
			}
			httpmock.RegisterResponder(
				"GET",
				fmt.Sprintf("https://esi.evetech.net/characters/%d/assets?page=1", c.ID),
				httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
					"is_singleton":  tc.singleton,
					"item_id":       itemID,
					"location_flag": "Hangar",
					"location_id":   60002959,
					"location_type": "station",
					"quantity":      tc.quantity,
					"type_id":       et.ID,
				}}).HeaderSet(http.Header{"X-Pages": []string{"1"}}),
			)
			httpmock.RegisterResponder(
				"POST",
				fmt.Sprintf("https://esi.evetech.net/characters/%d/assets/names", c.ID),
				httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
			)
			// when
			_, err := s.updateAssetsESI(ctx, characterSectionUpdateParams{
				characterID: c.ID,
				section:     app.SectionCharacterAssets,
			})
			// then
			require.NoError(t, err)
			x, err := st.GetCharacterAsset(ctx, c.ID, itemID)
			require.NoError(t, err)
			xassert.Equal(t, tc.singleton, x.IsSingleton)
			xassert.Equal(t, tc.wantQuantity, x.Quantity)
		})
	}
}

func TestUpdateCharacterAssetsESI_Name(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	const itemID = 1000000016835
	named := httpmock.NewJsonResponderOrPanic(200, []map[string]any{{"item_id": itemID, "name": "My Ship"}})
	unnamed := httpmock.NewJsonResponderOrPanic(200, []map[string]any{{"item_id": itemID, "name": "None"}})
	failed := httpmock.NewJsonResponderOrPanic(400, map[string]any{"error": "failed"})
	for _, tc := range []struct {
		name      string
		singleton bool
		names     httpmock.Responder
		want      string
	}{
		{name: "keeps name of singleton", singleton: true, names: named, want: "My Ship"},
		{name: "clears name when no longer singleton", singleton: false, names: named, want: ""},
		{name: "clears name when removed in game", singleton: true, names: unnamed, want: ""},
		{name: "keeps name when request fails", singleton: true, names: failed, want: "My Ship"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			c := factory.CreateCharacterFull()
			factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
			category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
			group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID})
			et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
			factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
			factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
				CharacterID: c.ID,
				IsSingleton: true,
				ItemID:      itemID,
				Name:        "My Ship",
				Quantity:    1,
				TypeID:      et.ID,
			})
			httpmock.RegisterResponder(
				"GET",
				fmt.Sprintf("https://esi.evetech.net/characters/%d/assets?page=1", c.ID),
				httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
					"is_singleton":  tc.singleton,
					"item_id":       itemID,
					"location_flag": "Hangar",
					"location_id":   60002959,
					"location_type": "station",
					"quantity":      1,
					"type_id":       et.ID,
				}}).HeaderSet(http.Header{"X-Pages": []string{"1"}}),
			)
			httpmock.RegisterResponder(
				"POST",
				fmt.Sprintf("https://esi.evetech.net/characters/%d/assets/names", c.ID),
				tc.names,
			)
			// when
			_, err := s.updateAssetsESI(ctx, characterSectionUpdateParams{
				characterID: c.ID,
				section:     app.SectionCharacterAssets,
			})
			// then
			require.NoError(t, err)
			x, err := st.GetCharacterAsset(ctx, c.ID, itemID)
			require.NoError(t, err)
			xassert.Equal(t, tc.singleton, x.IsSingleton)
			xassert.Equal(t, 60002959, x.LocationID)
			xassert.Equal(t, tc.want, x.Name)
		})
	}
}

func TestUpdateCharacterAssetsESI_NameFetchCanceled(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	var logs bytes.Buffer
	defer slog.SetDefault(slog.Default())
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const itemID = 1000000016835
	c := factory.CreateCharacterFull()
	factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
	category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
	group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID})
	et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
	factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
	factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
		CharacterID: c.ID,
		IsSingleton: true,
		ItemID:      itemID,
		Name:        "My Ship",
		Quantity:    1,
		TypeID:      et.ID,
	})
	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("https://esi.evetech.net/characters/%d/assets?page=1", c.ID),
		httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
			"is_singleton":  true,
			"item_id":       itemID,
			"location_flag": "Hangar",
			"location_id":   60002959,
			"location_type": "station",
			"quantity":      1,
			"type_id":       et.ID,
		}}).HeaderSet(http.Header{"X-Pages": []string{"1"}}),
	)
	httpmock.RegisterResponder(
		"POST",
		fmt.Sprintf("https://esi.evetech.net/characters/%d/assets/names", c.ID),
		func(*http.Request) (*http.Response, error) {
			cancel()
			return nil, context.Canceled
		},
	)
	// when
	_, err := s.updateAssetsESI(ctx, characterSectionUpdateParams{
		characterID: c.ID,
		section:     app.SectionCharacterAssets,
	})
	// then
	require.ErrorIs(t, err, context.Canceled)
	assert.NotContains(t, logs.String(), "Failed to fetch asset names")
	x, err := st.GetCharacterAsset(context.Background(), c.ID, itemID)
	require.NoError(t, err)
	xassert.Equal(t, "My Ship", x.Name)
}

func TestFetchAssetNamesESI(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	const ownerID = 42
	ids := make([]int64, 1000) // needs two requests
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	url := fmt.Sprintf("https://esi.evetech.net/characters/%d/assets/names", ownerID)
	ok := httpmock.NewJsonResponderOrPanic(200, []map[string]any{
		{"item_id": 1, "name": "Alpha"},
		{"item_id": 2, "name": "None"},
	})
	t.Run("returns all IDs when all requests succeed", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url, ok)
		got, err := s.fetchAssetNamesESI(ctx, ownerID, ids)
		require.NoError(t, err)
		assert.Len(t, got, 1000)
		xassert.Equal(t, "Alpha", got[1])
		xassert.Equal(t, "", got[2])
		xassert.Equal(t, "", got[1000])
	})
	t.Run("leaves out IDs of failed request and reports error", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url, ok.Then(
			httpmock.NewJsonResponderOrPanic(400, map[string]any{"error": "failed"}),
		))
		got, err := s.fetchAssetNamesESI(ctx, ownerID, ids)
		require.Error(t, err)
		assert.Len(t, got, 999)
		xassert.Equal(t, "Alpha", got[1])
		assert.NotContains(t, got, int64(1000))
	})
}

func TestUpdateAssetValue(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
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

		// when
		err := s.updateAssetValue(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		c2, err := st.GetCharacter(t.Context(), c.ID)
		require.NoError(t, err)
		got := c2.AssetValue
		assert.Equal(t, optional.New(500.5), got)
	})

}

func TestAssetTotalValue(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	t.Run("can return asset total value for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter(storage.CreateCharacterParams{AssetValue: optional.New(123.45)})

		// when
		got, err := s.AssetTotalValue(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		assert.Equal(t, optional.New(123.45), got)
	})
}

func TestListAssets(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	t.Run("can list assets for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		ca := factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		factory.CreateCharacterAsset() // asset for another character

		// when
		got, err := s.ListAssets(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		if assert.Len(t, got, 1) {
			assert.Equal(t, ca.ItemID, got[0].ItemID)
		}
	})
}

func TestListAllAssets(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	t.Run("can list assets across all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacterAsset()
		factory.CreateCharacterAsset()

		// when
		got, err := s.ListAllAssets(t.Context())

		// then
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}
