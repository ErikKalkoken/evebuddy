package http_test

import (
	"bytes"
	"log"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	ihttp "example/evebuddy/internal/helper/http"
)

func TestTransport(t *testing.T) {
	httpClient := &http.Client{
		Transport: ihttp.LoggedTransport{},
	}
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	t.Run("can log GET request with 200", func(t *testing.T) {
		// given
		slog.SetLogLoggerLevel(slog.LevelInfo)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(200, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, 200, r.StatusCode)
			assert.Contains(t, buf.String(), "INFO HTTP response method=GET url=https://www.example.com/ status=200")
		}
	})
	t.Run("can log POST request with 404", func(t *testing.T) {
		// given
		slog.SetLogLoggerLevel(slog.LevelInfo)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(404, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, 404, r.StatusCode)
			assert.Contains(t, buf.String(), "WARN HTTP response method=GET url=https://www.example.com/ status=404")
		}
	})
	t.Run("can log request and response details when level is DEBUG", func(t *testing.T) {
		// given
		slog.SetLogLoggerLevel(slog.LevelDebug)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(200, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, 200, r.StatusCode)
			assert.Contains(t, buf.String(), "INFO HTTP response method=GET url=https://www.example.com/ status=200")
			assert.Contains(t, buf.String(), "DEBUG HTTP request method=GET url=https://www.example.com/ header=map[] body=")
			assert.Contains(t, buf.String(), "DEBUG HTTP response method=GET url=https://www.example.com/ status=200 header=map[] body=Test")
		}
	})
}
