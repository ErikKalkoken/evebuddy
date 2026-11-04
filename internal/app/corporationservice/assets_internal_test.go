package corporationservice

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
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestUpdateCorporationAssetsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new assets from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
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
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets?page=1", c.ID),
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
			}).HeaderSet(http.Header{"X-Pages": []string{"1"}}),
		)
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets/names", c.ID),
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
		changed, err := s.updateAssetsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationAssets,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCorporationAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 2, ids.Size())
		x, err := st.GetCorporationAsset(ctx, c.ID, 1000000016835)
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
		x, err = st.GetCorporationAsset(ctx, c.ID, 1000000016836)
		require.NoError(t, err)
		xassert.Equal(t, "", x.Name)
	})
	t.Run("should remove obsolete items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID, ItemID: 1000000019999,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets?page=1", c.ID),
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
			}).HeaderSet(http.Header{"X-Pages": []string{"1"}}),
		)
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets/names", c.ID),
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
		changed, err := s.updateAssetsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationAssets,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCorporationAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1000000016835, 1000000016836), ids)
	})
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
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
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets?page=1", c.ID),
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
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets?page=2", c.ID),
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
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}))
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets/names", c.ID),
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
		changed, err := s.updateAssetsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationAssets,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCorporationAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 2, ids.Size())
		x, err := st.GetCorporationAsset(ctx, c.ID, 1000000016835)
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
		x, err = st.GetCorporationAsset(ctx, c.ID, 1000000016836)
		require.NoError(t, err)
		xassert.Equal(t, "", x.Name)
	})
}

func TestUpdateCorporationAssetsESI_Singleton(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
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
			s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
				AccessToken: "accessToken",
			}}})
			c := factory.CreateCorporation()
			et := factory.CreateEveType()
			factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
			if tc.existing {
				factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
					CorporationID: c.ID,
					EveTypeID:     et.ID,
					IsSingleton:   tc.existingSingleton,
					ItemID:        itemID,
					Quantity:      tc.existingQuantity,
				})
			}
			httpmock.RegisterResponder(
				"GET",
				fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets?page=1", c.ID),
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
				fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets/names", c.ID),
				httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
			)
			// when
			_, err := s.updateAssetsESI(ctx, corporationSectionUpdateParams{
				corporationID: c.ID,
				section:       app.SectionCorporationAssets,
			})
			// then
			require.NoError(t, err)
			x, err := st.GetCorporationAsset(ctx, c.ID, itemID)
			require.NoError(t, err)
			xassert.Equal(t, tc.singleton, x.IsSingleton)
			xassert.Equal(t, tc.wantQuantity, x.Quantity)
		})
	}
}

func TestUpdateCorporationAssetsESI_Name(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
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
			s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
				AccessToken: "accessToken",
			}}})
			c := factory.CreateCorporation()
			category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
			group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID})
			et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
			factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
			factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
				CorporationID: c.ID,
				EveTypeID:     et.ID,
				IsSingleton:   true,
				ItemID:        itemID,
				Name:          "My Ship",
				Quantity:      1,
			})
			httpmock.RegisterResponder(
				"GET",
				fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets?page=1", c.ID),
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
				fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets/names", c.ID),
				tc.names,
			)
			// when
			_, err := s.updateAssetsESI(ctx, corporationSectionUpdateParams{
				corporationID: c.ID,
				section:       app.SectionCorporationAssets,
			})
			// then
			require.NoError(t, err)
			x, err := st.GetCorporationAsset(ctx, c.ID, itemID)
			require.NoError(t, err)
			xassert.Equal(t, tc.singleton, x.IsSingleton)
			xassert.Equal(t, 60002959, x.LocationID)
			xassert.Equal(t, tc.want, x.Name)
		})
	}
}

func TestUpdateCorporationAssetsESI_NameFetchCanceled(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
		AccessToken: "accessToken",
	}}})
	var logs bytes.Buffer
	defer slog.SetDefault(slog.Default())
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const itemID = 1000000016835
	c := factory.CreateCorporation()
	category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
	group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID})
	et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
	factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
	factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
		CorporationID: c.ID,
		EveTypeID:     et.ID,
		IsSingleton:   true,
		ItemID:        itemID,
		Name:          "My Ship",
		Quantity:      1,
	})
	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets?page=1", c.ID),
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
		fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets/names", c.ID),
		func(*http.Request) (*http.Response, error) {
			cancel()
			return nil, context.Canceled
		},
	)
	// when
	_, err := s.updateAssetsESI(ctx, corporationSectionUpdateParams{
		corporationID: c.ID,
		section:       app.SectionCorporationAssets,
	})
	// then
	require.ErrorIs(t, err, context.Canceled)
	assert.NotContains(t, logs.String(), "Failed to fetch asset names")
	x, err := st.GetCorporationAsset(context.Background(), c.ID, itemID)
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
	url := fmt.Sprintf("https://esi.evetech.net/corporations/%d/assets/names", ownerID)
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

func TestListAssets(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can list assets and filter out alliance assets", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		et := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAlliance})
		a1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c.ID, EveTypeID: et.ID})
		got, err := s.ListAssets(ctx, c.ID)
		require.NoError(t, err)
		ids := set.Collect(xiter.MapSlice(got, func(x *app.CorporationAsset) int64 { return x.ItemID }))
		xassert.Equal(t, set.Of(a1.ItemID), ids)
	})
}

func TestListAllAssets(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can list assets from all corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		a1 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c1.ID})
		a2 := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{CorporationID: c2.ID})
		got, err := s.ListAllAssets(ctx)
		require.NoError(t, err)
		ids := set.Collect(xiter.MapSlice(got, func(x *app.CorporationAsset) int64 { return x.ItemID }))
		xassert.Equal(t, set.Of(a1.ItemID, a2.ItemID), ids)
	})
}

func TestCalculateAssetTotalValue(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can calculate total asset value for corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		ca := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			Quantity:      2,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       ca.Type.ID,
			AveragePrice: optional.New(100.0),
		})
		got, err := s.CalculateAssetTotalValue(ctx, c.ID)
		require.NoError(t, err)
		assert.InDelta(t, 200.0, got, 0.01)
	})
}

func TestCalculateAssetValueByDivision(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()

	t.Run("should sum asset value per hangar division and exclude non-division assets", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		loc := factory.CreateEveLocationStructure()

		et1 := factory.CreateEveType()
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       et1.ID,
			AveragePrice: optional.New(10.0),
		})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			EveTypeID:     et1.ID,
			LocationID:    loc.ID,
			LocationFlag:  app.FlagCorpSAG1,
			Quantity:      2,
		})

		et2 := factory.CreateEveType()
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       et2.ID,
			AveragePrice: optional.New(5.0),
		})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			EveTypeID:     et2.ID,
			LocationID:    loc.ID,
			LocationFlag:  app.FlagCorpSAG2,
			Quantity:      4,
		})

		et3 := factory.CreateEveType()
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       et3.ID,
			AveragePrice: optional.New(1000.0),
		})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			EveTypeID:     et3.ID,
			LocationID:    loc.ID,
			LocationFlag:  app.FlagHangar,
			Quantity:      1,
		})

		got, err := s.CalculateAssetValueByDivision(ctx, c.ID)

		require.NoError(t, err)
		assert.InDelta(t, 20.0, got[app.Division1], 0.01)
		assert.InDelta(t, 20.0, got[app.Division2], 0.01)
		assert.NotContains(t, got, app.Division3)
	})

	t.Run("should include value of items nested inside a container within a division", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		loc := factory.CreateEveLocationStructure()
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID})
		shipType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		container := factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			EveTypeID:     shipType.ID,
			LocationID:    loc.ID,
			LocationFlag:  app.FlagCorpSAG1,
			IsSingleton:   true,
		})
		cargoType := factory.CreateEveType()
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       cargoType.ID,
			AveragePrice: optional.New(50.0),
		})
		factory.CreateCorporationAsset(storage.CreateCorporationAssetParams{
			CorporationID: c.ID,
			EveTypeID:     cargoType.ID,
			LocationID:    container.ItemID,
			LocationFlag:  app.FlagCargo,
			Quantity:      3,
		})

		got, err := s.CalculateAssetValueByDivision(ctx, c.ID)

		require.NoError(t, err)
		assert.InDelta(t, 150.0, got[app.Division1], 0.01)
	})

	t.Run("should return empty map when corporation has no assets", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()

		got, err := s.CalculateAssetValueByDivision(ctx, c.ID)

		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestAssets_AdoptNames(t *testing.T) {
	assets := []*app.CorporationAsset{
		{
			Asset: app.Asset{
				ItemID: 1,
				Type:   &app.EveType{ID: 2233},
			},
		},
		{
			Asset: app.Asset{
				ItemID: 2,
				Type:   &app.EveType{ID: 2233},
			},
		},
		{
			Asset: app.Asset{
				ItemID: 3,
				Type:   &app.EveType{ID: 42},
			},
		},
		{
			Asset: app.Asset{
				ItemID: 4,
				Type:   &app.EveType{ID: 2233},
			},
		},
	}
	names := map[int64]string{
		1: "Customs Office (Sirikur VII)",
		3: "Alpha",
		4: "Bravo",
	}

	modifyAssetNames(assets, names)

	xassert.Equal(t, "Sirikur VII", names[1])
	assert.NotContains(t, names, 2)
	xassert.Equal(t, "Alpha", names[3])
	xassert.Equal(t, "Bravo", names[4])
}
