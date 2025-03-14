package main

import (
	"bytes"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
)

type myLogger struct {
	retryablehttp.Logger
}

func TestLogResponse(t *testing.T) {
	t.Run("should log request and response details when log level is DEBUG", func(t *testing.T) {
		// given
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer func() {
			log.SetOutput(os.Stderr)
		}()

		rhc := retryablehttp.NewClient()
		rhc.Logger = slog.Default()
		rhc.ResponseLogHook = logResponse
		slog.SetLogLoggerLevel(slog.LevelDebug)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("dummy", "alpha")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`orange`))
		}))
		defer server.Close()

		req, err := retryablehttp.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}

		// when
		r, err := rhc.Do(req)

		// then
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Conditionf(t, func() bool {
				m, err := regexp.MatchString(`DEBUG HTTP response method=GET .*status=200.*header=\".*Dummy:\[alpha\].*\".*body=orange`, buf.String())
				if err != nil {
					t.Fatal(err)
				}
				return m
			}, "buffer: %s", buf.String())
		}
	})
	t.Run("should not log response details when log level is not DEBUG", func(t *testing.T) {
		// given
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer func() {
			log.SetOutput(os.Stderr)
		}()

		rhc := retryablehttp.NewClient()
		rhc.Logger = slog.Default()
		rhc.ResponseLogHook = logResponse
		slog.SetLogLoggerLevel(slog.LevelInfo)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("dummy", "alpha")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`orange`))
		}))
		defer server.Close()

		req, err := retryablehttp.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}

		// when
		r, err := rhc.Do(req)

		// then
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.NotContains(t, buf.String(), "HTTP response")
		}
	})
}
