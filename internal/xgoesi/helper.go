package xgoesi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// createErrorResponse creates a synthetic response for an HTTP error.
func createErrorResponse(req *http.Request, statusCode int, retryAfterSeconds int, message string) (*http.Response, error) {
	if statusCode < 400 {
		return nil, fmt.Errorf("statusCode must be >=400")
	}
	if retryAfterSeconds <= 0 {
		return nil, fmt.Errorf("retryAfterSeconds must be >0")
	}
	data, err := json.Marshal(map[string]string{"error": message})
	if err != nil {
		return nil, err
	}
	var statusText string
	if statusCode == 420 {
		statusText = "Too Many Errors"
	} else {
		statusText = http.StatusText(statusCode)
	}
	resp := &http.Response{
		Status:     fmt.Sprintf("%d %s", statusCode, statusText),
		StatusCode: statusCode,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header: map[string][]string{
			"Retry-After":     {strconv.Itoa(retryAfterSeconds)},
			"X-Origin-Server": {"localhost"},
		},
		ContentLength: int64(len(data)),
		Request:       req,
	}
	resp.Header.Set("Content-Type", "application/json")
	return resp, nil
}
