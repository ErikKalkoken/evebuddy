package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestMarketOrderStateString(t *testing.T) {
	cases := []struct {
		s    app.MarketOrderState
		want string
	}{
		{app.OrderCancelled, "cancelled"},
		{app.OrderExpired, "expired"},
		{app.OrderOpen, "open"},
		{app.OrderUndefined, "undefined"},
		{app.OrderUnknown, "unknown"},
		{app.MarketOrderState(99), "?"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.String())
		})
	}
}
