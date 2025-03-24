package characterservice

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestFetchFromESIWithPaging(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	ctx := context.Background()
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		httpmock.Reset()
		pages := "3"
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v5/characters/99/assets/",
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
			"https://esi.evetech.net/v5/characters/99/assets/?page=2",
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
			"GET",
			"https://esi.evetech.net/v5/characters/99/assets/?page=3",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"is_blueprint_copy": true,
					"is_singleton":      false,
					"item_id":           1000000016837,
					"location_flag":     "Hangar",
					"location_id":       60002959,
					"location_type":     "station",
					"quantity":          1,
					"type_id":           3516,
				},
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}))
		// when
		xx, err := fetchFromESIWithPaging(
			func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
				arg := &esi.GetCharactersCharacterIdAssetsOpts{
					Page: esioptional.NewInt32(int32(pageNum)),
				}
				return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
			})
		// then
		if assert.NoError(t, err) {
			assert.Len(t, xx, 3)
			want := []int64{1000000016835, 1000000016836, 1000000016837}
			got := make([]int64, 3)
			for i, x := range xx {
				got[i] = x.ItemId
			}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should fetch single page", func(t *testing.T) {
		// given
		httpmock.Reset()
		pages := "1"
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v5/characters/99/assets/",
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
		// when
		xx, err := fetchFromESIWithPaging(
			func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
				arg := &esi.GetCharactersCharacterIdAssetsOpts{
					Page: esioptional.NewInt32(int32(pageNum)),
				}
				return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
			})
		// then
		if assert.NoError(t, err) {
			assert.Len(t, xx, 1)
		}
	})
	t.Run("can ignore missing x-pages header", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v5/characters/99/assets/",
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
			}))
		// when
		xx, err := fetchFromESIWithPaging(
			func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
				arg := &esi.GetCharactersCharacterIdAssetsOpts{
					Page: esioptional.NewInt32(int32(pageNum)),
				}
				return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
			})
		// then
		if assert.NoError(t, err) {
			assert.Len(t, xx, 1)
		}
	})
	t.Run("should return error from function", func(t *testing.T) {
		// given
		myErr := errors.New("error")
		// when
		_, err := fetchFromESIWithPaging(
			func(pageNum int) ([]int, *http.Response, error) {
				return nil, nil, myErr
			})
		// then
		assert.ErrorIs(t, err, myErr)
	})
	t.Run("should return error when X-Pages is invalid", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v5/characters/99/assets/",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}).HeaderSet(http.Header{"X-Pages": []string{"invalid"}}))
		// when
		_, err := fetchFromESIWithPaging(
			func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
				arg := &esi.GetCharactersCharacterIdAssetsOpts{
					Page: esioptional.NewInt32(int32(pageNum)),
				}
				return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
			})
		// then
		assert.Error(t, err)
	})
}
