package esi

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// A mail list a character has subscribed to
type MailList struct {
	ID   int32  `json:"mailing_list_id"`
	Name string `json:"name"`
}

// FetchMailLists fetches a character's subscribed mail lists from ESI and returns them.
func FetchMailLists(httpClient *http.Client, characterID int32, tokenString string) ([]MailList, error) {
	v := url.Values{}
	v.Set("token", tokenString)
	fullUrl := fmt.Sprintf("%s/characters/%d/mail/lists/?%v", esiBaseUrl, characterID, v.Encode())
	log.Printf("Fetching mail lists for character ID %d", characterID)
	resp, err := httpClient.Get(fullUrl)
	if err != nil {
		return nil, err
	}

	m, err := unmarshalResponse[[]MailList](resp)
	return m, err
}
