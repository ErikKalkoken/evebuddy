package esi

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// A generic error returned from ESI
type ESIError struct {
	Error string `json:"error"`
}

func getESI(c http.Client, path string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", esiBaseUrl, path)
	maxRetries := 3
	retries := 0
	for {
		r, err := c.Get(url)
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

	if err := json.Unmarshal(body, &o); err != nil {
		return o, fmt.Errorf("%v: %v", err, string(body))
	}
	return o, nil
}
