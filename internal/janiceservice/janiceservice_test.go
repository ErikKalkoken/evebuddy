package janiceservice_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	t.Run("should panic when trying to create without client", func(t *testing.T) {
		assert.Panics(t, func() {
			janiceservice.New(nil, "abc")
		})
	})
}

func TestPricer(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Run("should return price info", func(t *testing.T) {
		data := map[string]any{
			"date": "2025-04-25T01:02:03Z",
			"market": map[string]any{
				"id":   2,
				"name": "Jita 4-4",
			},
			"buyOrderCount":  33,
			"buyVolume":      11113209302,
			"sellOrderCount": 81,
			"sellVolume":     11504901017,
			"immediatePrices": map[string]any{
				"buyPrice":              4.04,
				"splitPrice":            4.045,
				"sellPrice":             4.05,
				"buyPrice5DayMedian":    3.9979999999999998,
				"splitPrice5DayMedian":  4.045,
				"sellPrice5DayMedian":   4.07,
				"buyPrice30DayMedian":   4.058666666666666,
				"splitPrice30DayMedian": 4.085,
				"sellPrice30DayMedian":  4.15,
			},
			"top5AveragePrices": map[string]any{
				"buyPrice":              3.968120198672043,
				"splitPrice":            4.018029513197122,
				"sellPrice":             4.067938827722203,
				"buyPrice5DayMedian":    3.968120198672043,
				"splitPrice5DayMedian":  4.018029513197122,
				"sellPrice5DayMedian":   4.083768770864654,
				"buyPrice30DayMedian":   4.05,
				"splitPrice30DayMedian": 4.090959141867867,
				"sellPrice30DayMedian":  4.171827042133677,
			},
			"itemType": map[string]any{
				"eid":            34,
				"name":           "Tritanium",
				"volume":         0.01,
				"packagedVolume": 0.02,
			},
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://janice.e-351.com/api/rest/v2/pricer/34",
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		s := janiceservice.New(http.DefaultClient, "api-key")
		x, err := s.FetchPrices(context.Background(), 34)
		if assert.NoError(t, err) {
			assert.Equal(t, time.Date(2025, 4, 25, 1, 2, 3, 0, time.UTC), x.Date)
			assert.EqualValues(t, 2, x.Market.ID)
			assert.EqualValues(t, "Jita 4-4", x.Market.Name)
			assert.EqualValues(t, 11113209302, x.BuyVolume)
			assert.EqualValues(t, 11504901017, x.SellVolume)
			assert.InDelta(t, 4.04, x.ImmediatePrices.BuyPrice, 0.005)
			assert.InDelta(t, 3.97, x.Top5AveragePrices.BuyPrice, 0.005)
			assert.EqualValues(t, 34, x.ItemType.EID)
			assert.EqualValues(t, "Tritanium", x.ItemType.Name)
			assert.InDelta(t, 0.01, x.ItemType.Volume, 0.005)
			assert.InDelta(t, 0.02, x.ItemType.PackagedVolume, 0.005)
		}
	})
	t.Run("should return HTTP error", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://janice.e-351.com/api/rest/v2/pricer/34",
			httpmock.NewJsonResponderOrPanic(404, map[string]any{}),
		)
		s := janiceservice.New(http.DefaultClient, "api-key")
		_, err := s.FetchPrices(context.Background(), 34)
		assert.ErrorIs(t, err, janiceservice.ErrHttpError)
	})
	t.Run("should return error when called with invalid type ID", func(t *testing.T) {
		s := janiceservice.New(http.DefaultClient, "api-key")
		_, err := s.FetchPrices(context.Background(), 0)
		assert.Error(t, err)
	})
	t.Run("should return error when no API key", func(t *testing.T) {
		s := janiceservice.New(http.DefaultClient, "")
		_, err := s.FetchPrices(context.Background(), 34)
		assert.Error(t, err)
	})
}
