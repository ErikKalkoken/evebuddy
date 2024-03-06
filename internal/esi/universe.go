package esi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
func ResolveEntityIDs(httpClient *http.Client, ids []int32) ([]EveEntity, error) {
	if len(ids) > 1000 {
		return nil, fmt.Errorf("too many ids")
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}

	fullUrl := fmt.Sprintf("%s/universe/names/", esiBaseUrl)
	log.Printf("Resolving IDs from %v", fullUrl)
	resp, err := httpClient.Post(fullUrl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	return unmarshalResponse[[]EveEntity](resp)
}
