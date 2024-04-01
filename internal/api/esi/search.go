package esi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// An Eve entity returned from ESI.
type SearchResult struct {
	Agent         []int32 `json:"agent"`
	Alliance      []int32 `json:"alliance"`
	Character     []int32 `json:"character"`
	Constellation []int32 `json:"constellation"`
	Corporation   []int32 `json:"corporation"`
	Faction       []int32 `json:"faction"`
	InventoryType []int32 `json:"inventory_type"`
	Region        []int32 `json:"region"`
	SolarSystem   []int32 `json:"solar_system"`
	Station       []int32 `json:"station"`
	Structure     []int32 `json:"structure"`
}

// Search makes a search request to ESI and returns the results
func Search(client *http.Client, characterID int32, search string, tokenString string) (*SearchResult, error) {
	v := url.Values{"categories": {"character"}}
	v.Set("search", search)
	p := fmt.Sprintf("/characters/%d/search/?%v", characterID, v.Encode())
	r, err := raiseError(getESIWithToken(client, p, tokenString))
	if err != nil {
		return nil, err
	}
	var s SearchResult
	if err := json.Unmarshal(r.body, &s); err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(r.body))
	}
	return &s, err
}
