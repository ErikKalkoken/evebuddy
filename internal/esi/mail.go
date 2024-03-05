package esi

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type MailRecipient struct {
	ID   int32  `json:"recipient_id"`
	Type string `json:"recipient_type"`
}

type MailHeader struct {
	FromID     int32           `json:"from"`
	IsRead     bool            `json:"is_read"`
	ID         int32           `json:"mail_id"`
	Labels     []int32         `json:"labels"`
	Recipients []MailRecipient `json:"recipients"`
	Subject    string          `json:"subject"`
	Timestamp  string          `json:"timestamp"`
}

func FetchMailHeaders(characterID int32, tokenString string) ([]MailHeader, error) {
	v := url.Values{}
	v.Set("token", tokenString)
	fullUrl := fmt.Sprintf("%s/characters/%d/mail/?%v", esiBaseUrl, characterID, v.Encode())
	log.Printf("Fetching mail headers for %d", characterID)
	resp, err := http.Get(fullUrl)
	if err != nil {
		return nil, err
	}

	return UnmarshalResponse[[]MailHeader](resp)
}

type Mail struct {
	MailHeader
	Body string `json:"body"`
}

func FetchMail(characterID int32, mailID int32, tokenString string) (*Mail, error) {
	v := url.Values{}
	v.Set("token", tokenString)
	fullUrl := fmt.Sprintf("%s/characters/%d/mail/%d/?%v", esiBaseUrl, characterID, mailID, v.Encode())
	log.Printf("Fetching mail with ID %d for %d", mailID, characterID)
	resp, err := http.Get(fullUrl)
	if err != nil {
		return nil, err
	}

	m, err := UnmarshalResponse[Mail](resp)
	return &m, err
}
