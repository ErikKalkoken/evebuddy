package esi

import (
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
func ResolveEntityIDs(client *http.Client, ids []int32) ([]EveEntity, error) {
	if len(ids) > 1000 {
		return nil, fmt.Errorf("too many ids")
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	slog.Info("Request to resolve IDs", "count", len(ids))
	slog.Debug("IDs to resolve", "ids", ids)
	body, err := postESI(client, "/universe/names/", data)
	if err != nil {
		return nil, err
	}
	var ee []EveEntity
	if err := json.Unmarshal(body, &ee); err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(body))
	}
	return ee, err

}
