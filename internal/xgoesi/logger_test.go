package xgoesi_test

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func TestCanceledDowngradeLogger(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	rhc := retryablehttp.NewClient()
	rhc.Logger = xgoesi.CanceledDowngradeLogger{Logger: slog.Default()}
	rhc.RetryMax = 0
	t.Run("should not log canceled request at Info level", func(t *testing.T) {
		// given
		logBuf.Reset()
		slog.SetLogLoggerLevel(slog.LevelInfo)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, "https://www.example.com/", nil)
		if !assert.NoError(t, err) {
			return
		}

		// when
		_, err = rhc.Do(req)

		// then
		assert.Error(t, err)
		assert.NotContains(t, logBuf.String(), "request failed")
		assert.NotContains(t, logBuf.String(), "ERROR")
	})
	t.Run("should log canceled request at Debug level when enabled", func(t *testing.T) {
		// given
		logBuf.Reset()
		slog.SetLogLoggerLevel(slog.LevelDebug)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, "https://www.example.com/", nil)
		if !assert.NoError(t, err) {
			return
		}

		// when
		_, err = rhc.Do(req)

		// then
		assert.Error(t, err)
		assert.Contains(t, logBuf.String(), "DEBUG")
		assert.Contains(t, logBuf.String(), "request failed")
		assert.NotContains(t, logBuf.String(), "ERROR")
	})
	t.Run("should still log non-canceled request failures at Error level", func(t *testing.T) {
		// given
		logBuf.Reset()
		slog.SetLogLoggerLevel(slog.LevelInfo)
		httpmock.ActivateNonDefault(rhc.HTTPClient)
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder(
			http.MethodGet,
			"https://www.example.com/",
			httpmock.NewErrorResponder(assert.AnError),
		)
		req, err := retryablehttp.NewRequest(http.MethodGet, "https://www.example.com/", nil)
		if !assert.NoError(t, err) {
			return
		}

		// when
		_, err = rhc.Do(req)

		// then
		assert.Error(t, err)
		assert.Contains(t, logBuf.String(), "ERROR")
		assert.Contains(t, logBuf.String(), "request failed")
	})
}
