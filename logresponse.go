package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	headerContentTypeKey  = "Content-Type"
	headerContentTypeJSON = "application/json"
)

// Responses from these URLs will never be logged.
var blacklistedURLs = []string{"login.eveonline.com/v2/oauth/token"}

// logResponse is a callback for retryablehttp.
// It logs all HTTP errors and also the complete response when log level is DEBUG.
func logResponse(l retryablehttp.Logger, r *http.Response) {
	isDebug := slog.Default().Enabled(context.Background(), slog.LevelDebug)
	isHTTPError := r.StatusCode >= 400
	if !isDebug && !isHTTPError {
		return
	}

	var level slog.Level
	if isHTTPError {
		level = slog.LevelWarn
	} else {
		level = slog.LevelDebug

	}

	data, err := extractBodyForLog(r)
	if err != nil {
		slog.Error("Failed to extract response body", "error", err)
		data = nil
	}

	status := statusText(r)
	var args []any
	if isDebug {
		args = []any{
			"method", r.Request.Method,
			"url", r.Request.URL,
			"status", status,
			"header", r.Header,
			"body", data,
		}
	} else {
		args = []any{
			"method", r.Request.Method,
			"url", r.Request.URL,
			"status", status,
			"body", data,
		}
	}

	slog.Log(context.Background(), level, "HTTP response", args...)
}

func extractBodyForLog(r *http.Response) (any, error) {
	x := r.Header.Get(headerContentTypeKey)
	var parts []string
	for _, s := range strings.Split(x, ";") {
		parts = append(parts, strings.Trim(s, " "))
	}
	isJSON := slices.Contains(parts, headerContentTypeJSON)
	hasBlacklistedURL := slices.ContainsFunc(blacklistedURLs, func(x string) bool {
		return strings.Contains(r.Request.URL.String(), x)
	})
	if hasBlacklistedURL {
		if !isJSON {
			return "xxxxx", nil
		}
		return map[string]bool{"redacted": true}, nil
	}
	body, err := copyResponseBody(r)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}
	if !isJSON {
		return string(body), nil
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// copyResponseBody returns a copy of the response body r. It preserves the body.
func copyResponseBody(r *http.Response) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

// statusText returns the status code of a response with adding information.
func statusText(r *http.Response) string {
	var s string
	if r.StatusCode == 420 {
		s = "Error Limited"
	} else {
		s = http.StatusText(r.StatusCode)
	}
	return fmt.Sprintf("%d %s", r.StatusCode, s)
}
