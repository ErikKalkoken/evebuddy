package esi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

// A mail list a character has subscribed to
type MailList struct {
	ID   int32  `json:"mailing_list_id"`
	Name string `json:"name"`
}

// FetchMailLists fetches a character's subscribed mail lists from ESI and returns them.
func FetchMailLists(client *http.Client, characterID int32, tokenString string) ([]MailList, error) {
	v := url.Values{}
	v.Set("token", tokenString)
	path := fmt.Sprintf("/characters/%d/mail/lists/?%v", characterID, v.Encode())
	r, err := raiseError(getESI(client, path))
	if err != nil {
		return nil, err
	}
	var mm []MailList
	if err := json.Unmarshal(r.body, &mm); err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(r.body))
	}
	slog.Info("Received mail lists from ESI", "characterID", characterID, "count", len(mm))
	return mm, err
}
