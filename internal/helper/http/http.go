// Package http contains HTTP related helpers
package http

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const maxRetries = 3

// TODO: Add tests

// LoggedTransport adds request slog logging.
//
// Responses with status code below 400 are logged with INFO level.
// Responses with status code of 400 or higher are logged with WARNING level.
// When DEBUG logging is enabled, will also log details of request and response.
type LoggedTransport struct{}

func (r LoggedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	isDebug := logRequest(req)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	logResponse(isDebug, resp, req)
	return resp, err
}

// ESITransport adds request logging and automatic retrying for common ESI HTTP errors
type ESITransport struct{}

func (r ESITransport) RoundTrip(req *http.Request) (*http.Response, error) {
	isDebug := logRequest(req)
	retry := 0
	for {
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			return resp, err
		}
		logResponse(isDebug, resp, req)
		if (resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusGatewayTimeout || resp.StatusCode == http.StatusServiceUnavailable) && retry < maxRetries {
			retry++
			slog.Warn("Retrying", "method", req.Method, "url", req.URL, "retry", retry, "maxRetries", maxRetries)
			wait := time.Millisecond * time.Duration(200*retry)
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
		slog.Debug("HTTP request", "method", req.Method, "url", req.URL, "header", req.Header, "body", reqBody)
	}
	return isDebug
}

func logResponse(isDebug bool, resp *http.Response, req *http.Request) {
	if isDebug {
		respBody := ""
		if resp.Body != nil {
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				respBody = string(body)
				resp.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}
		slog.Debug("HTTP response", "method", req.Method, "url", req.URL, "status", resp.StatusCode, "header", resp.Header, "body", respBody)
	}
	var level slog.Level
	if resp.StatusCode >= 400 {
		level = slog.LevelWarn
	} else {
		level = slog.LevelInfo
	}
	slog.Log(context.Background(), level, "HTTP response", "method", req.Method, "url", req.URL, "status", resp.StatusCode)
}
