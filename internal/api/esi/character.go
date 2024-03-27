package esi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Character struct {
	AllianceID    int32  `json:"alliance_id"`
	CorporationID int32  `json:"corporation_id"`
	FactionID     int32  `json:"faction_id"`
	Name          string `json:"name"`
}

func FetchCharacter(client *http.Client, characterID int32) (*Character, error) {
	path := fmt.Sprintf("/characters/%d/", characterID)
	slog.Info("Fetching character", "ID", characterID)
	body, err := getESI(client, path)
	if err != nil {
		return nil, err
	}
	var c Character
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(body))
	}
	return &c, err
}
