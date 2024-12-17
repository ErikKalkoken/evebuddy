// Package http provides custom http transport implementations.
package httptransport

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"
)

// LoggedTransport adds request slog logging.
//
// Responses with status code below 400 are logged with INFO level.
// Responses with status code of 400 or higher are logged with WARNING level.
// When DEBUG logging is enabled, will also log details of request and response including headers.
// Authorization headers in requests are redacted.
// Can redact response bodies for URLs (e.g. which would contain tokens)
type LoggedTransport struct {
	// Body of blacklisted response URLs will not be logged.
	BlacklistedResponseURLs []string
}

func (t LoggedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	isDebug := logRequest(req)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	logResponse(t.BlacklistedResponseURLs, isDebug, resp, req)
	return resp, err
}

// defaults for LoggedTransportWithRetries
const (
	defaultMaxRetries        = 3
	defaultDelayMilliseconds = 200
)

// LoggedTransportWithRetries adds request logging and automatic retrying for common HTTP errors.
type LoggedTransportWithRetries struct {
	MaxRetries         int
	StatusCodesToRetry []int
	DelayMilliseconds  int
	// Pattern for blocked URLs. Body of blocked URLs will not be logged.
	BlockedResponseURLs []string
}

func (t LoggedTransportWithRetries) RoundTrip(req *http.Request) (*http.Response, error) {
	isDebug := logRequest(req)
	maxRetries := t.MaxRetries
	if maxRetries < 1 {
		maxRetries = defaultMaxRetries
	}
	delayMilliseconds := t.DelayMilliseconds
	if delayMilliseconds < 1 {
		delayMilliseconds = defaultDelayMilliseconds
	}
	retry := 0
	for {
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			return resp, err
		}
		logResponse(t.BlockedResponseURLs, isDebug, resp, req)
		if slices.Contains(t.StatusCodesToRetry, resp.StatusCode) && retry < maxRetries {
			retry++
			slog.Warn("Retrying", "method", req.Method, "url", req.URL, "retry", retry, "maxRetries", maxRetries)
			wait := time.Duration(delayMilliseconds*retry) * time.Millisecond
			time.Sleep(wait)
			continue
		}
		return resp, err
	}
}

func logRequest(req *http.Request) bool {
	isDebug := slog.Default().Enabled(context.Background(), slog.LevelDebug)
	if isDebug {
		reqBody := ""
		if req.Body != nil {
			body, err := io.ReadAll(req.Body)
			if err == nil {
				reqBody = string(body)
				req.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}
		h := req.Header.Clone()
		if h.Get("Authorization") != "" {
			h.Set("Authorization", "REDACTED") // never log this header
		}
		slog.Debug("HTTP request", "method", req.Method, "url", req.URL, "header", h, "body", reqBody)
	}
	return isDebug
}

func logResponse(blockedURLs []string, isDebug bool, resp *http.Response, req *http.Request) {
	if isDebug {
		var respBody string
		var isBlocked bool
		url := req.URL.String()
		for _, u := range blockedURLs {
			if strings.Contains(url, u) {
				isBlocked = true
				break
			}
		}
		if isBlocked {
			respBody = "REDACTED"
		} else if resp.Body != nil {
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				respBody = string(body)
				resp.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}
		slog.Debug(
			"HTTP response",
			"method", req.Method,
			"url", req.URL,
			"status",
			resp.StatusCode,
			"header",
			resp.Header,
			"body",
			respBody,
		)
	}
	var level slog.Level
	if resp.StatusCode >= 400 {
		level = slog.LevelWarn
	} else {
		level = slog.LevelInfo
	}
	slog.Log(
		context.Background(),
		level,
		"HTTP response",
		"method",
		req.Method,
		"url",
		req.URL,
		"status",
		resp.StatusCode,
	)
}
