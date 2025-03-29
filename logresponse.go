package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

// Resonses from these URLs will never be logged.
var blacklistedURLs = []string{"login.eveonline.com/v2/oauth/token"}

// logResponse is a callback for retryable logger, which is called for every respose.
// It logs all HTTP erros and also the complete response when log level is DEBUG.
func logResponse(l retryablehttp.Logger, r *http.Response) {
	isDebug := slog.Default().Enabled(context.Background(), slog.LevelDebug)
	isHttpError := r.StatusCode >= 400
	if !isDebug && !isHttpError {
		return
	}

	var level slog.Level
	if isHttpError {
		level = slog.LevelWarn
	} else {
		level = slog.LevelDebug

	}
	status := statusText(r)
	body := bodyToString(r)
	var args []any
	if isDebug {
		args = []any{
			"method", r.Request.Method,
			"url", r.Request.URL,
			"status", status,
			"header", r.Header,
			"body", body,
		}
	} else {
		args = []any{
			"method", r.Request.Method,
			"url", r.Request.URL,
			"status", status,
			"body", body,
		}
	}

	slog.Log(context.Background(), level, "HTTP response", args...)
}

func bodyToString(r *http.Response) string {
	if r.Body == nil {
		return ""
	}
	hasBlockedURL := slices.ContainsFunc(blacklistedURLs, func(x string) bool {
		return strings.Contains(r.Request.URL.String(), x)
	})
	if hasBlockedURL {
		return "xxxxx"
	}
	var s string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s = "ERROR: " + err.Error()
	} else {
		s = string(body)
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return s
}

func statusText(r *http.Response) string {
	var s string
	if r.StatusCode == 420 {
		s = "Error Limited"
	} else {
		s = http.StatusText(r.StatusCode)
	}
	return fmt.Sprintf("%d %s", r.StatusCode, s)
}
