package xesi

import (
	"context"
	"errors"
	"net/http"
	"testing"

	esioptional "github.com/antihax/goesi/optional"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestFetchPagesConcurrently(t *testing.T) {
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
			"https://esi.evetech.net/v4/characters/99/assets/",
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
			"https://esi.evetech.net/v4/characters/99/assets/?page=2",
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
			"GET",
			"https://esi.evetech.net/v4/characters/99/assets/?page=3",
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
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		// when
		xx, err := FetchPagesConcurrently(-1, func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
			arg := &esi.GetCharactersCharacterIdAssetsOpts{
				Page: esioptional.NewInt32(int32(pageNum)),
			}
			return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
		})
		// then
		if assert.NoError(t, err) {
			want := []int64{1000000016835, 1000000016836, 1000000016837}
			got := make([]int64, 0)
			for _, x := range xx {
				got = append(got, x.ItemId)
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
			"https://esi.evetech.net/v4/characters/99/assets/",
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
		// when
		xx, err := FetchPagesConcurrently(-1,
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
			"https://esi.evetech.net/v4/characters/99/assets/",
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
			}),
		)
		// when
		xx, err := FetchPagesConcurrently(-1,
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
		_, err := FetchPagesConcurrently(-1,
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
			"https://esi.evetech.net/v4/characters/99/assets/",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}).HeaderSet(http.Header{"X-Pages": []string{"invalid"}}))
		// when
		_, err := FetchPagesConcurrently(-1,
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

func TestFetchPagesWithShortcut(t *testing.T) {
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
			"https://esi.evetech.net/v4/characters/99/assets/",
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
			"https://esi.evetech.net/v4/characters/99/assets/?page=2",
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
			"GET",
			"https://esi.evetech.net/v4/characters/99/assets/?page=3",
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
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		// when
		xx, err := FetchPagesWithExit(func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
			arg := &esi.GetCharactersCharacterIdAssetsOpts{
				Page: esioptional.NewInt32(int32(pageNum)),
			}
			return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
		}, nil)
		// then
		if assert.NoError(t, err) {
			want := []int64{1000000016835, 1000000016836, 1000000016837}
			got := make([]int64, 0)
			for _, x := range xx {
				got = append(got, x.ItemId)
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
			"https://esi.evetech.net/v4/characters/99/assets/",
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
		// when
		xx, err := FetchPagesWithExit(func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
			arg := &esi.GetCharactersCharacterIdAssetsOpts{
				Page: esioptional.NewInt32(int32(pageNum)),
			}
			return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
		}, nil)
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
			"https://esi.evetech.net/v4/characters/99/assets/",
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
			}),
		)
		// when
		xx, err := FetchPagesWithExit(func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
			arg := &esi.GetCharactersCharacterIdAssetsOpts{
				Page: esioptional.NewInt32(int32(pageNum)),
			}
			return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
		}, nil)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, xx, 1)
		}
	})
	t.Run("should return error from function", func(t *testing.T) {
		// given
		myErr := errors.New("error")
		// when
		_, err := FetchPagesWithExit(func(pageNum int) ([]int, *http.Response, error) {
			return nil, nil, myErr
		}, nil)
		// then
		assert.ErrorIs(t, err, myErr)
	})
	t.Run("should return error when X-Pages is invalid", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v4/characters/99/assets/",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}).HeaderSet(http.Header{"X-Pages": []string{"invalid"}}))
		// when
		_, err := FetchPagesWithExit(func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
			arg := &esi.GetCharactersCharacterIdAssetsOpts{
				Page: esioptional.NewInt32(int32(pageNum)),
			}
			return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
		}, nil)
		// then
		assert.Error(t, err)
	})
	t.Run("should fetch pages until exit function returns true", func(t *testing.T) {
		// given
		httpmock.Reset()
		pages := "3"
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v4/characters/99/assets/",
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
			"https://esi.evetech.net/v4/characters/99/assets/?page=2",
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
			"GET",
			"https://esi.evetech.net/v4/characters/99/assets/?page=3",
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
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		// when
		xx, err := FetchPagesWithExit(func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
			arg := &esi.GetCharactersCharacterIdAssetsOpts{
				Page: esioptional.NewInt32(int32(pageNum)),
			}
			return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
		}, func(x esi.GetCharactersCharacterIdAssets200Ok) bool {
			return x.ItemId == 1000000016836
		})
		// then
		if assert.NoError(t, err) {
			want := []int64{1000000016835, 1000000016836}
			got := make([]int64, 0)
			for _, x := range xx {
				got = append(got, x.ItemId)
			}
			assert.Equal(t, want, got)
		}
	})
	t.Run("can exit after first page", func(t *testing.T) {
		// given
		httpmock.Reset()
		pages := "3"
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v4/characters/99/assets/",
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
			"https://esi.evetech.net/v4/characters/99/assets/?page=2",
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
			"GET",
			"https://esi.evetech.net/v4/characters/99/assets/?page=3",
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
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		// when
		xx, err := FetchPagesWithExit(func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
			arg := &esi.GetCharactersCharacterIdAssetsOpts{
				Page: esioptional.NewInt32(int32(pageNum)),
			}
			return client.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, 99, arg)
		}, func(x esi.GetCharactersCharacterIdAssets200Ok) bool {
			return x.ItemId == 1000000016835
		})
		// then
		if assert.NoError(t, err) {
			want := []int64{1000000016835}
			got := make([]int64, 0)
			for _, x := range xx {
				got = append(got, x.ItemId)
			}
			assert.Equal(t, want, got)
		}
	})
}
