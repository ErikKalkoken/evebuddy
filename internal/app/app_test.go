package app_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"testing"

	"github.com/fnt-eve/goesi-openapi"
	"github.com/jarcoal/httpmock"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEntityShort_IDOrZero(t *testing.T) {
	x1 := &app.EntityShort{ID: 42, Name: "Alpha"}
	xassert.Equal(t, 42, x1.IDOrZero())
	var x2 *app.EntityShort
	xassert.Equal(t, 0, x2.IDOrZero())
}

func TestEntityShort_NameOrZero(t *testing.T) {
	x1 := &app.EntityShort{ID: 42, Name: "Alpha"}
	xassert.Equal(t, "Alpha", x1.NameOrZero())
	var x2 *app.EntityShort
	xassert.Equal(t, "", x2.NameOrZero())
}

// fakeNetError is a minimal net.Error implementation for testing.
type fakeNetError struct {
	msg       string
	timeout   bool
	temporary bool
}

func (e fakeNetError) Error() string   { return e.msg }
func (e fakeNetError) Timeout() bool   { return e.timeout }
func (e fakeNetError) Temporary() bool { return e.temporary }

func TestError(t *testing.T) {
	t.Run("should return no error for nil", func(t *testing.T) {
		got := app.ErrorDisplay(nil)
		xassert.Equal(t, "No error", got)
	})
	t.Run("should return general error for an unrecognized error", func(t *testing.T) {
		err := errors.New("new error")
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "general error", got)
	})
	t.Run("should return general error for a wrapped unrecognized error", func(t *testing.T) {
		err := fmt.Errorf("context: %w", errors.New("new error"))
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "general error", got)
	})
	t.Run("should recognize a token error", func(t *testing.T) {
		got := app.ErrorDisplay(app.ErrTokenError)
		xassert.Equal(t, "token error", got)
	})
	t.Run("should recognize a wrapped token error", func(t *testing.T) {
		err := fmt.Errorf("context: %w", app.ErrTokenError)
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "token error", got)
	})
	t.Run("should recognize a database error", func(t *testing.T) {
		err := sqlite3.Error{}
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "database error", got)
	})
	t.Run("should recognize a wrapped database error", func(t *testing.T) {
		err := fmt.Errorf("context: %w", sqlite3.Error{})
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "database error", got)
	})
	t.Run("should recognize a dial network error", func(t *testing.T) {
		err := &net.OpError{Op: "dial", Err: errors.New("boom")}
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "unknown host", got)
	})
	t.Run("should recognize a wrapped dial network error", func(t *testing.T) {
		err := fmt.Errorf("context: %w", &net.OpError{Op: "dial", Err: errors.New("boom")})
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "unknown host", got)
	})
	t.Run("should recognize a read network error", func(t *testing.T) {
		err := &net.OpError{Op: "read", Err: errors.New("boom")}
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "connection refused", got)
	})
	t.Run("should recognize an other network op error", func(t *testing.T) {
		err := &net.OpError{Op: "write", Err: errors.New("boom")}
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "network error", got)
	})
	t.Run("should recognize a connection refused errno", func(t *testing.T) {
		got := app.ErrorDisplay(syscall.ECONNREFUSED)
		xassert.Equal(t, "connection refused", got)
	})
	t.Run("should recognize a wrapped connection refused errno", func(t *testing.T) {
		err := fmt.Errorf("context: %w", syscall.ECONNREFUSED)
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "connection refused", got)
	})
	t.Run("should return general error for another errno", func(t *testing.T) {
		got := app.ErrorDisplay(syscall.EINVAL)
		xassert.Equal(t, "general error", got)
	})
	t.Run("should recognize a url error", func(t *testing.T) {
		err := &url.Error{Op: "Get", URL: "https://example.com", Err: errors.New("boom")}
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "network error", got)
	})
	t.Run("should recognize a wrapped url error", func(t *testing.T) {
		err := fmt.Errorf("context: %w", &url.Error{Op: "Get", URL: "https://example.com", Err: errors.New("boom")})
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "network error", got)
	})
	t.Run("should recognize a timeout net error", func(t *testing.T) {
		err := fakeNetError{msg: "boom", timeout: true}
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "timeout", got)
	})
	t.Run("should recognize a wrapped timeout net error", func(t *testing.T) {
		err := fmt.Errorf("context: %w", fakeNetError{msg: "boom", timeout: true})
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "timeout", got)
	})
	t.Run("should recognize a non-timeout net error", func(t *testing.T) {
		err := fakeNetError{msg: "boom", timeout: false}
		got := app.ErrorDisplay(err)
		xassert.Equal(t, "network error", got)
	})
	t.Run("should resolve goesi errors", func(t *testing.T) {
		// given
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(400, map[string]any{
				"error": "my error",
			}),
		)
		client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
			UserAgent: "MyApp/1.0 (contact@example.com)",
		})
		ctx := context.Background()
		_, _, err := client.MarketAPI.GetMarketsPrices(ctx).Execute()
		require.Error(t, err)
		// when
		got := app.ErrorDisplay(err)
		// then
		xassert.Equal(t, "400 Bad Request: my error", got)
	})
	t.Run("should resolve wrapped goesi errors", func(t *testing.T) {
		// given
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(400, map[string]any{
				"error": "my error",
			}),
		)
		client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
			UserAgent: "MyApp/1.0 (contact@example.com)",
		})
		ctx := context.Background()
		_, _, err := client.MarketAPI.GetMarketsPrices(ctx).Execute()
		require.Error(t, err)
		// when
		got := app.ErrorDisplay(fmt.Errorf("context: %w", err))
		// then
		xassert.Equal(t, "400 Bad Request: my error", got)
	})
}
