package xgoesi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
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

// LogResponse is a callback for retryablehttp.
// It logs all HTTP errors and also the complete response when log level is DEBUG.
func LogResponse(_ retryablehttp.Logger, r *http.Response) {
	if r == nil {
		return
	}
	var ctx context.Context
	if r.Request != nil {
		ctx = r.Request.Context()
	} else {
		ctx = context.Background()
	}

	isDebug := slog.Default().Enabled(ctx, slog.LevelDebug)
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
	reqMethod := ""
	reqURL := ""
	if r.Request != nil {
		reqMethod = r.Request.Method
		if r.Request.URL != nil {
			reqURL = r.Request.URL.String()
		}
	}
	args := []any{
		"method", reqMethod,
		"url", reqURL,
		"status", status,
	}
	if isDebug {
		args = append(args, "header", r.Header)
	}
	args = append(args, "body", data)

	slog.Log(ctx, level, "HTTP response", args...)
}

func extractBodyForLog(r *http.Response) (any, error) {
	ct := r.Header.Get(headerContentTypeKey)
	mediaType, _, _ := mime.ParseMediaType(ct)
	isJSON := mediaType == headerContentTypeJSON

	var reqURL string
	if r.Request != nil && r.Request.URL != nil {
		reqURL = r.Request.URL.String()
	}
	hasBlacklistedURL := slices.ContainsFunc(blacklistedURLs, func(x string) bool {
		return reqURL != "" && strings.Contains(reqURL, x)
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
	if r.StatusCode == StatusTooManyErrors {
		s = "Error Limited"
	} else {
		s = http.StatusText(r.StatusCode)
	}
	return fmt.Sprintf("%d %s", r.StatusCode, s)
}
