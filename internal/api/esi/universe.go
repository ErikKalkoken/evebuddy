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
	slog.Debug("Trying to resolve IDs", "ids", ids)
	r, err := raiseError(postESI(client, "/universe/names/", data))
	if err != nil {
		return nil, err
	}
	var ee []EveEntity
	if err := json.Unmarshal(r.body, &ee); err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(r.body))
	}
	slog.Info("Received resolved entities from ESI", "count", len(ee))
	return ee, err
}

type UniverseIDsEntry struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// Response from universe/ids/
type UniverseIDsResponse struct {
	Agents         []UniverseIDsEntry `json:"agents"`
	Alliances      []UniverseIDsEntry `json:"alliances"`
	Characters     []UniverseIDsEntry `json:"characters"`
	Constellations []UniverseIDsEntry `json:"constellations"`
	Corporations   []UniverseIDsEntry `json:"corporations"`
	Factions       []UniverseIDsEntry `json:"factions"`
	InventoryTypes []UniverseIDsEntry `json:"inventory_types"`
	Regions        []UniverseIDsEntry `json:"regions"`
	SolarSystems   []UniverseIDsEntry `json:"solar_systems"`
	Stations       []UniverseIDsEntry `json:"stations"`
}

func ResolveEntityNames(client *http.Client, names []string) (*UniverseIDsResponse, error) {
	if len(names) > 500 {
		return nil, fmt.Errorf("too many names")
	}
	data, err := json.Marshal(names)
	if err != nil {
		return nil, err
	}
	slog.Debug("Trying to resolve names", "names", names)
	r, err := raiseError(postESI(client, "/universe/ids/", data))
	if err != nil {
		return nil, err
	}
	var result UniverseIDsResponse
	if err := json.Unmarshal(r.body, &result); err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(r.body))
	}
	return &result, err
}
