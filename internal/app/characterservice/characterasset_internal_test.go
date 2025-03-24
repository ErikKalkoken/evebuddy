package characterservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

func TestUpdateCharacterAssetsESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should create new assets from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516})
		location := factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/assets/", c.ID),
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
			}).HeaderSet(http.Header{"X-Pages": []string{"1"}}))
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/assets/names/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"item_id": 1000000016835,
					"name":    "Awesome Name",
				},
				{
					"item_id": 1000000016836,
					"name":    "None",
				},
			}))
		// when
		changed, err := s.updateCharacterAssetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionAssets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 2)
				x, err := st.GetCharacterAsset(ctx, c.ID, 1000000016835)
				if assert.NoError(t, err) {
					assert.Equal(t, eveType.ID, x.EveType.ID)
					assert.Equal(t, eveType.Name, x.EveType.Name)
					assert.True(t, x.IsBlueprintCopy)
					assert.True(t, x.IsSingleton)
					assert.Equal(t, "Hangar", x.LocationFlag)
					assert.Equal(t, location.ID, x.LocationID)
					assert.Equal(t, "station", x.LocationType)
					assert.Equal(t, "Awesome Name", x.Name)
					assert.Equal(t, int32(1), x.Quantity)
				}
				x, err = st.GetCharacterAsset(ctx, c.ID, 1000000016836)
				if assert.NoError(t, err) {
					assert.Equal(t, "", x.Name)
				}
			}
		}
	})
	t.Run("should remove obsolete items", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516})
		factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID, ItemID: 1000000019999,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/assets/", c.ID),
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
			}).HeaderSet(http.Header{"X-Pages": []string{"1"}}))
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/assets/names/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"item_id": 1000000016835,
					"name":    "Awesome Name",
				},
				{
					"item_id": 1000000016836,
					"name":    "None",
				},
			}))
		// when
		changed, err := s.updateCharacterAssetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionAssets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, []int64{1000000016835, 1000000016836}, ids.ToSlice())
			}
		}
	})
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516})
		location := factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		pages := "2"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/assets/", c.ID),
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
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/assets/?page=2", c.ID),
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
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/assets/names/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"item_id": 1000000016835,
					"name":    "Awesome Name",
				},
				{
					"item_id": 1000000016836,
					"name":    "None",
				},
			}))
		// when
		changed, err := s.updateCharacterAssetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionAssets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 2)
				x, err := st.GetCharacterAsset(ctx, c.ID, 1000000016835)
				if assert.NoError(t, err) {
					assert.Equal(t, eveType.ID, x.EveType.ID)
					assert.Equal(t, eveType.Name, x.EveType.Name)
					assert.True(t, x.IsBlueprintCopy)
					assert.True(t, x.IsSingleton)
					assert.Equal(t, "Hangar", x.LocationFlag)
					assert.Equal(t, location.ID, x.LocationID)
					assert.Equal(t, "station", x.LocationType)
					assert.Equal(t, "Awesome Name", x.Name)
					assert.Equal(t, int32(1), x.Quantity)
				}
				x, err = st.GetCharacterAsset(ctx, c.ID, 1000000016836)
				if assert.NoError(t, err) {
					assert.Equal(t, "", x.Name)
				}
			}
		}
	})
}

func newCharacterService(st *storage.Storage) *CharacterService {
	sc := statuscache.New(memcache.New())
	eu := eveuniverseservice.New(st, nil)
	eu.StatusCacheService = sc
	s := New(st, nil, nil)
	s.EveUniverseService = eu
	s.StatusCacheService = sc
	return s
}
