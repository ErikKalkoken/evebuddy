package esi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// An Eve entity returned from ESI.
type EveEntity struct {
	Category string `json:"category"`
	ID       int32  `json:"id"`
	Name     string `json:"name"`
}

// TODO: Should work for more then 1000 IDs
// TODO: Add smart handling for unsolvable IDs

// ResolveEntityIDs tries to resolve IDs to Eve entity objects and returns them.
func ResolveEntityIDs(httpClient http.Client, ids []int32) ([]EveEntity, error) {
	if len(ids) > 1000 {
		return nil, fmt.Errorf("too many ids")
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf("%s/universe/names/", esiBaseUrl)
	slog.Info("Request to resolve IDs", "url", fullURL, "count", len(ids))
	slog.Debug("IDs to resolve", "ids", ids)
	resp, err := httpClient.Post(fullURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	return unmarshalResponse[[]EveEntity](resp)
}
