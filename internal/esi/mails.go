package esi

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// A mail returned from ESI.
type Mail struct {
	MailHeader
	Body string `json:"body"`
}

// FetchMail fetches a mail for a character from ESI and returns it.
func FetchMail(client http.Client, characterID int32, mailID int32, tokenString string) (*Mail, error) {
	v := url.Values{}
	v.Set("token", tokenString)
	path := fmt.Sprintf("/characters/%d/mail/%d/?%v", characterID, mailID, v.Encode())
	log.Printf("Fetching mail with ID %d for %d", mailID, characterID)
	resp, err := getESI(client, path)
	if err != nil {
		return nil, err
	}

	m, err := unmarshalResponse[Mail](resp)
	return &m, err
}
