package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	ctx := context.Background()
	t.Run("should return general error", func(t *testing.T) {
		err := errors.New("new error")
		got := app.ErrorDisplay(err)
		assert.Equal(t, "general error", got)
	})
	t.Run("should resolve goesi errors", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/markets/prices/",
			httpmock.NewJsonResponderOrPanic(400, map[string]any{
				"error": "my error",
			}))
		_, _, err := client.ESI.MarketApi.GetMarketsPrices(ctx, nil)
		// when
		got := app.ErrorDisplay(err)
		assert.Equal(t, "400 Bad Request: my error", got)
	})
}
