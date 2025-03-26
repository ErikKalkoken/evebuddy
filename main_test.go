package main

import (
	"bytes"
	"log"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

type myLogger struct {
	retryablehttp.Logger
}

func TestLogResponse(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	rhc := retryablehttp.NewClient()
	rhc.Logger = slog.Default()
	rhc.ResponseLogHook = logResponse
	httpmock.ActivateNonDefault(rhc.HTTPClient)
	defer httpmock.DeactivateAndReset()
	t.Run("should log request and response details when log level is DEBUG", func(t *testing.T) {
		// given
		logBuf.Reset()
		slog.SetLogLoggerLevel(slog.LevelDebug)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusOK, "orange").HeaderSet(http.Header{"dummy": []string{"alpha"}}))

		// when
		r, err := rhc.Get("https://www.example.com/")

		// then
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Conditionf(t, func() bool {
				m, err := regexp.MatchString(
					`DEBUG HTTP response method=GET .*status="200.*header=.*Dummy:\[alpha\].*.*body=orange`,
					logBuf.String(),
				)
				if err != nil {
					t.Fatal(err)
				}
				return m
			}, logBuf.String())
		}
	})
	t.Run("should not log response details when log level is not DEBUG and no HTTP error", func(t *testing.T) {
		// given
		logBuf.Reset()
		slog.SetLogLoggerLevel(slog.LevelInfo)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusOK, "orange").HeaderSet(http.Header{"dummy": []string{"alpha"}}))

		// when
		r, err := rhc.Get("https://www.example.com/")

		// then
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.NotContains(t, logBuf.String(), "HTTP response")
		}
	})
	t.Run("should log response warning when HTTP error and include body", func(t *testing.T) {
		// given
		logBuf.Reset()
		slog.SetLogLoggerLevel(slog.LevelInfo)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://www.example.com/",
			httpmock.NewStringResponder(http.StatusNotFound, "orange").HeaderSet(http.Header{"dummy": []string{"alpha"}}))

		// when
		r, err := rhc.Get("https://www.example.com/")

		// then
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusNotFound, r.StatusCode)
			assert.Conditionf(t, func() bool {
				m, err := regexp.MatchString(`WARN HTTP response .*body=`, logBuf.String())
				if err != nil {
					t.Fatal(err)
				}
				return m
			}, logBuf.String())
		}
	})
	t.Run("should redact response body for blacklisted URLs", func(t *testing.T) {
		// given
		logBuf.Reset()
		slog.SetLogLoggerLevel(slog.LevelDebug)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://login.eveonline.com/v2/oauth/token",
			httpmock.NewStringResponder(http.StatusOK, "orange"))

		// when
		r, err := rhc.Get("https://login.eveonline.com/v2/oauth/token")

		// then
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, r.StatusCode)
			assert.Conditionf(t, func() bool {
				m, err := regexp.MatchString(`DEBUG HTTP response .*body=xxxxx`, logBuf.String())
				if err != nil {
					t.Fatal(err)
				}
				return m
			}, logBuf.String())
		}
	})
}
