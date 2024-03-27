package esi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

// A mail returned from ESI.
type Mail struct {
	MailHeader
	Body string `json:"body"`
}

// FetchMail fetches a mail for a character from ESI and returns it.
func FetchMail(client *http.Client, characterID int32, mailID int32, tokenString string) (*Mail, error) {
	v := url.Values{}
	v.Set("token", tokenString)
	path := fmt.Sprintf("/characters/%d/mail/%d/?%v", characterID, mailID, v.Encode())
	slog.Info("Fetching mail for character", "mailID", mailID, "characterID", characterID)
	body, err := getESI(client, path)
	if err != nil {
		return nil, err
	}
	var m Mail
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(body))
	}
	return &m, err
}
