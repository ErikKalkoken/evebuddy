package esi

import (
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

func FetchCharacter(client http.Client, characterID int32) (*Character, error) {
	path := fmt.Sprintf("/characters/%d/", characterID)
	slog.Info("Fetching character", "ID", characterID)
	resp, err := getESI(client, path)
	if err != nil {
		return nil, err
	}
	c, err := unmarshalResponse[Character](resp)
	if err != nil {
		return nil, err
	}
	return &c, err
}
