package esi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// A generic error returned from ESI
type EsiError struct {
	Error string `json:"error"`
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
