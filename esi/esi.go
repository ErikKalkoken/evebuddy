package esi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

const (
	esiBaseUrl = "https://esi.evetech.net/latest"
)

type CharacterContact struct {
	ContactId   int32   `json:"contact_id"`
	ContactType string  `json:"contact_type"`
	Standing    float32 `json:"standing"`
}

func FetchContacts(characterID int32, tokenString string) []CharacterContact {
	v := url.Values{}
	v.Set("token", tokenString)
	fullUrl := fmt.Sprintf("%s/characters/%d/contacts/?%v", esiBaseUrl, characterID, v.Encode())
	log.Printf("Fetching contacts from %v", fullUrl)
	resp, err := http.Get(fullUrl)
	if err != nil {
		log.Fatal(err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var contacts []CharacterContact
	if err := json.Unmarshal(body, &contacts); err != nil {
		log.Fatal(err)
	}
	return contacts
}
