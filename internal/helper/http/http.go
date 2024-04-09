// Package http contains HTTP related helpers
package http

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
)

type CustomTransport struct{}

func (r CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	isDebug := logRequest(req)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	logResponse(isDebug, resp, req)
	return resp, err
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
		slog.Debug("HTTP response", "method", req.Method, "url", req.URL, "status", resp.StatusCode, "body", respBody)
	}
	var level slog.Level
	if resp.StatusCode >= 400 {
		level = slog.LevelWarn
	} else {
		level = slog.LevelInfo
	}
	slog.Log(context.Background(), level, "HTTP response", "method", req.Method, "url", req.URL, "status", resp.StatusCode)
}
