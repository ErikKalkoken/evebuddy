package esi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type EveEntity struct {
	Category string `json:"category"`
	ID       int32  `json:"id"`
	Name     string `json:"name"`
}

func ResolveEntityIDs(ids []int32) ([]EveEntity, error) {
	if len(ids) > 1000 {
		return nil, fmt.Errorf("too many ids")
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}

	fullUrl := fmt.Sprintf("%s/universe/names/", esiBaseUrl)
	log.Printf("Resolving IDs from %v", fullUrl)
	resp, err := http.Post(fullUrl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	return UnmarshalResponse[[]EveEntity](resp)
}
