package esi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// A generic error returned from ESI
type ESIError struct {
	Error string `json:"error"`
}

func getESI(c *http.Client, path string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", esiBaseUrl, path)
	for i := 0; i < 3; i++ {
		r, err := c.Get(url)
		if err != nil {
			return nil, err
		}
		if r.StatusCode < 400 {
			return r, nil
		}

		log.Printf("ESI returned error: %v", r.Status)
		if r.StatusCode == http.StatusBadGateway || r.StatusCode == http.StatusGatewayTimeout || r.StatusCode == http.StatusServiceUnavailable {
			continue // retry
		}
		return nil, fmt.Errorf("error %v", r.Status)
	}
	return nil, fmt.Errorf("too many retries")
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
