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

func TestLoggedTransport(t *testing.T) {
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
			httpmock.NewStringResponder(http.StatusOK, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
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
			httpmock.NewStringResponder(http.StatusNotFound, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusNotFound, r.StatusCode)
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
			httpmock.NewStringResponder(http.StatusOK, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Contains(t, buf.String(), "INFO HTTP response method=GET url=https://www.example.com/ status=200")
			assert.Contains(t, buf.String(), "DEBUG HTTP request method=GET url=https://www.example.com/ header=map[] body=")
			assert.Contains(t, buf.String(), "DEBUG HTTP response method=GET url=https://www.example.com/ status=200 header=map[] body=Test")
		}
	})
}

func TestLoggedTransportWithRetries(t *testing.T) {
	httpClient := &http.Client{
		Transport: ihttp.LoggedTransportWithRetries{
			MaxRetries:         3,
			StatusCodesToRetry: []int{http.StatusBadGateway},
			DelayMilliseconds:  10,
		},
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
			httpmock.NewStringResponder(http.StatusOK, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Equal(t, httpmock.GetTotalCallCount(), 1)
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
			httpmock.NewStringResponder(http.StatusNotFound, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusNotFound, r.StatusCode)
			assert.Equal(t, httpmock.GetTotalCallCount(), 1)
			assert.Contains(t, buf.String(), "WARN HTTP response method=GET url=https://www.example.com/ status=404")
		}
	})
	t.Run("can log request and response details when level is DEBUG", func(t *testing.T) {
		// given
		slog.SetLogLoggerLevel(slog.LevelDebug)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusOK, "Answer"))
		// when
		r, err := httpClient.Post("https://www.example.com/", "", bytes.NewBuffer([]byte("Question")))
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Equal(t, httpmock.GetTotalCallCount(), 1)
			assert.Contains(t, buf.String(), "INFO HTTP response method=POST url=https://www.example.com/ status=200")
			assert.Contains(t, buf.String(), "DEBUG HTTP request method=POST url=https://www.example.com/ header=map[Content-Type:[]] body=")
			assert.Contains(t, buf.String(), "DEBUG HTTP response method=POST url=https://www.example.com/ status=200 header=map[] body=Answer")
		}
	})
	t.Run("should retry on 502", func(t *testing.T) {
		// given
		slog.SetLogLoggerLevel(slog.LevelInfo)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusBadGateway, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusBadGateway, r.StatusCode)
			assert.Equal(t, httpmock.GetTotalCallCount(), 4)
		}
	})
	t.Run("should not retry on 500", func(t *testing.T) {
		// given
		slog.SetLogLoggerLevel(slog.LevelInfo)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusInternalServerError, "Test"))
		// when
		r, err := httpClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusInternalServerError, r.StatusCode)
			assert.Equal(t, httpmock.GetTotalCallCount(), 1)
		}
	})
}
