package app_test

import (
	"errors"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestError(t *testing.T) {
	t.Run("should return general error", func(t *testing.T) {
		err := errors.New("new error")
		got := app.ErrorDisplay(err)
	xassert.Equal(t, "general error", got)
	})
	// t.Run("should resolve goesi errors", func(t *testing.T) {
	// 	// given
	// 	httpmock.Activate()
	// 	defer httpmock.DeactivateAndReset()
	// 	httpmock.RegisterResponder(
	// 		"GET",
	// 		"https://esi.evetech.net/v1/markets/prices/",
	// 		httpmock.NewJsonResponderOrPanic(400, map[string]any{
	// 			"error": "my error",
	// 		}),
	// 	)
	// 	client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
	// 		UserAgent: "MyApp/1.0 (contact@example.com)",
	// 	})
	// 	ctx := context.Background()
	// 	_, _, err := client.MarketAPI.GetMarketsPrices(ctx).Execute()
	// 	// when
	// 	got := app.ErrorDisplay(err)
	// xassert.Equal(t, "400 Bad Request: my error", got)
	// })
}
