package esi

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type MailLabel struct {
	ID          int32  `json:"label_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	UnreadCount int32  `json:"unread_count"`
}

type MailLabelPayload struct {
	Labels           []MailLabel `json:"labels"`
	TotalUnreadCount int32       `json:"total_unread_count"`
}

// FetchMailLabels fetches a character's mail labels from ESI and returns them.
func FetchMailLabels(client http.Client, characterID int32, tokenString string) (*MailLabelPayload, error) {
	v := url.Values{}
	v.Set("token", tokenString)
	path := fmt.Sprintf("/characters/%d/mail/labels/?%v", characterID, v.Encode())
	log.Printf("Fetching mail labels for character ID %d", characterID)
	resp, err := getESI(client, path)
	if err != nil {
		return nil, err
	}
	m, err := unmarshalResponse[MailLabelPayload](resp)
	return &m, err
}
