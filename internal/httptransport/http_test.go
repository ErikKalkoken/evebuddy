package httptransport_test

import (
	"bytes"
	"log"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/httptransport"
)

func TestLoggedTransport(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	t.Run("can log GET request with 200", func(t *testing.T) {
		// given
		myClient := &http.Client{
			Transport: httptransport.LoggedTransport{},
		}
		slog.SetLogLoggerLevel(slog.LevelInfo)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusOK, "Test"))
		// when
		r, err := myClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Contains(t, buf.String(), "INFO HTTP response method=GET url=https://www.example.com/ status=200")
		}
	})
	t.Run("can log POST request with 404", func(t *testing.T) {
		// given
		myClient := &http.Client{
			Transport: httptransport.LoggedTransport{},
		}
		slog.SetLogLoggerLevel(slog.LevelInfo)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusNotFound, "Test"))
		// when
		r, err := myClient.Get("https://www.example.com/")
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusNotFound, r.StatusCode)
			assert.Contains(t, buf.String(), "WARN HTTP response method=GET url=https://www.example.com/ status=404")
		}
	})
	t.Run("can log request and response details when level is DEBUG", func(t *testing.T) {
		// given
		myClient := &http.Client{
			Transport: httptransport.LoggedTransport{},
		}
		slog.SetLogLoggerLevel(slog.LevelDebug)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusOK, "Test").HeaderSet(http.Header{"dummy": []string{"bravo"}}))
		req, err := http.NewRequest("GET", "https://www.example.com/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("dummy", "alpha")
		// when
		r, err := myClient.Do(req)
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Contains(t, buf.String(), "INFO HTTP response method=GET url=https://www.example.com/ status=200")
			assert.Contains(t, buf.String(), "DEBUG HTTP request method=GET url=https://www.example.com/ header=map[Dummy:[alpha]] body=")
			assert.Contains(t, buf.String(), "DEBUG HTTP response method=GET url=https://www.example.com/ status=200 header=map[Dummy:[bravo]] body=Test")
		}
	})
	t.Run("should never log authorization headers in request", func(t *testing.T) {
		// given
		myClient := &http.Client{
			Transport: httptransport.LoggedTransport{},
		}
		slog.SetLogLoggerLevel(slog.LevelDebug)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusOK, "Test"))
		req, err := http.NewRequest("GET", "https://www.example.com/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "token")
		req.Header.Set("Dummy", "alpha")
		// when
		r, err := myClient.Do(req)
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Contains(t, buf.String(), "DEBUG HTTP request method=GET url=https://www.example.com/ header=\"map[Authorization:[REDACTED] Dummy:[alpha]]\" body=")
		}
	})
	t.Run("can redact response bodies for blocked URLs", func(t *testing.T) {
		// given
		myClient := &http.Client{
			Transport: httptransport.LoggedTransport{
				BlockedResponseURLs: []string{"https://www.example.com/"},
			},
		}
		slog.SetLogLoggerLevel(slog.LevelDebug)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusOK, "Test"))
		req, err := http.NewRequest("GET", "https://www.example.com/", nil)
		if err != nil {
			t.Fatal(err)
		}
		// when
		r, err := myClient.Do(req)
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Contains(t, buf.String(), "DEBUG HTTP response method=GET url=https://www.example.com/ status=200 header=map[] body=REDACTED")
		}
	})
}

func TestLoggedTransportWithRetries(t *testing.T) {
	httpClient := &http.Client{
		Transport: httptransport.LoggedTransportWithRetries{
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
		// then
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
		// then
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
		// then
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
		// then
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
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusInternalServerError, r.StatusCode)
			assert.Equal(t, httpmock.GetTotalCallCount(), 1)
		}
	})
}
