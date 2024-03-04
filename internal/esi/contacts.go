package esi

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type CharacterContact struct {
	ContactId   int32   `json:"contact_id"`
	ContactType string  `json:"contact_type"`
	Standing    float32 `json:"standing"`
}

func FetchContacts(characterID int32, tokenString string) ([]CharacterContact, error) {
	v := url.Values{}
	v.Set("token", tokenString)
	fullUrl := fmt.Sprintf("%s/characters/%d/contacts/?%v", esiBaseUrl, characterID, v.Encode())
	log.Printf("Fetching contacts from %v", fullUrl)
	resp, err := http.Get(fullUrl)
	if err != nil {
		return nil, err
	}
	return UnmarshalResponse[[]CharacterContact](resp)
}
