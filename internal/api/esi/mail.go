package esi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// A mail returned from ESI.
type Mail struct {
	MailHeader
	Body string `json:"body"`
}

// FetchMail fetches a mail for a character from ESI and returns it.
func FetchMail(client *http.Client, characterID int32, mailID int32, tokenString string) (*Mail, error) {
	path := fmt.Sprintf("/characters/%d/mail/%d/", characterID, mailID)
	r, err := raiseError(getESIWithToken(client, path, tokenString))
	if err != nil {
		return nil, err
	}
	var m Mail
	if err := json.Unmarshal(r.body, &m); err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(r.body))
	}
	slog.Info("Received mail from ESI", "characterID", characterID, "mailID", mailID)
	return &m, err
}
