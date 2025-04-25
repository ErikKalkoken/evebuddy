package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"testing/iotest"

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

func TestBodyToString(t *testing.T) {
	t.Run("should return body", func(t *testing.T) {
		u, _ := url.Parse("http://www.example.com")
		r := &http.Response{
			Body: io.NopCloser(strings.NewReader("test")),
			Request: &http.Request{
				URL: u,
			},
		}
		x := bodyToString(r)
		assert.Equal(t, "test", x)
	})
	t.Run("should return empty when no body", func(t *testing.T) {
		u, _ := url.Parse("http://www.example.com")
		r := &http.Response{
			Request: &http.Request{
				URL: u,
			},
		}
		x := bodyToString(r)
		assert.Equal(t, "", x)
	})
	t.Run("should redact blocked URL", func(t *testing.T) {
		u, _ := url.Parse("https://login.eveonline.com/v2/oauth/token")
		r := &http.Response{
			Body: io.NopCloser(strings.NewReader("test")),
			Request: &http.Request{
				URL: u,
			},
		}
		x := bodyToString(r)
		assert.Equal(t, "xxxxx", x)
	})
	t.Run("can handle error in reading body", func(t *testing.T) {
		u, _ := url.Parse("http://www.example.com")
		b := io.NopCloser(iotest.ErrReader(errors.New("custom error")))
		r := &http.Response{
			Request: &http.Request{
				URL: u,
			},
			Body: b,
		}
		x := bodyToString(r)
		assert.Equal(t, "ERROR: custom error", x)
	})
}

func TestStatusText(t *testing.T) {
	t.Run("should return status text for normal codes", func(t *testing.T) {
		r := &http.Response{
			StatusCode: 200,
		}
		x := statusText(r)
		assert.Equal(t, "200 OK", x)
	})
	t.Run("should return status text for 420", func(t *testing.T) {
		r := &http.Response{
			StatusCode: 420,
		}
		x := statusText(r)
		assert.Equal(t, "420 Error Limited", x)
	})
}

func TestObfuscate(t *testing.T) {
	cases := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"normal", "123456789", 4, "XXXXX6789"},
		{"s too short", "123", 4, "XXX"},
		{"n is zero", "123456789", 0, "XXXXXXXXX"},
		{"n is negativ", "123456789", -5, "XXXXXXXXX"},
		{"s is empty", "", 4, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := obfuscate(tc.s, tc.n)
			assert.Equal(t, tc.want, got)
		})
	}
}
