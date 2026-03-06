package characterservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateCharacterAssetsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
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
