package esi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// A generic error returned from ESI
type esiError struct {
	Error string `json:"error"`
}

func getESI(c http.Client, path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, buildEsiUrl(path), nil)
	if err != nil {
		return nil, err
	}
	return sendRequest(c, req)
}

func postESI(c http.Client, path string, data []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, buildEsiUrl(path), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return sendRequest(c, req)
}

func buildEsiUrl(path string) string {
	u := fmt.Sprintf("%s%s", esiBaseUrl, path)
	return u
}
func sendRequest(c http.Client, req *http.Request) (*http.Response, error) {
	maxRetries := 3
	retries := 0
	for {
		slog.Info("Sending HTTP request", "method", req.Method, "url", req.URL)
		r, err := c.Do(req)
		if err != nil {
			return nil, err
		}
		if r.StatusCode == http.StatusOK {
			return r, nil
		}

		slog.Warn("ESI status response not OK", "status", r.Status)
		if r.StatusCode == http.StatusBadGateway || r.StatusCode == http.StatusGatewayTimeout || r.StatusCode == http.StatusServiceUnavailable {
			if retries < maxRetries {
				slog.Info("Retrying", "retries", retries, "maxRetries", maxRetries)
				retries++
				continue
			}
			return nil, fmt.Errorf("too many retries")
		}
		return nil, fmt.Errorf("error %v", r.Status)
	}
}

// unmarshalResponse converts a JSON response from ESI into an object.
func unmarshalResponse[T any](resp *http.Response) (T, error) {
	var o T
	if resp.Body != nil {
		defer resp.Body.Close()
	} else {
		return o, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return o, err
	}
	slog.Debug("ESI response", "body", string(body))
	if err := json.Unmarshal(body, &o); err != nil {
		return o, fmt.Errorf("%v: %v", err, string(body))
	}
	return o, nil
}
