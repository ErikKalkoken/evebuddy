package esi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type MailLabel struct {
	LabelID     int32  `json:"label_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	UnreadCount int32  `json:"unread_count"`
}

type MailLabelPayload struct {
	Labels           []MailLabel `json:"labels"`
	TotalUnreadCount int32       `json:"total_unread_count"`
}

// FetchMailLabels fetches a character's mail labels from ESI and returns them.
func FetchMailLabels(client *http.Client, characterID int32, tokenString string) (*MailLabelPayload, error) {
	path := fmt.Sprintf("/characters/%d/mail/labels/", characterID)
	r, err := raiseError(getESIWithToken(client, path, tokenString))
	if err != nil {
		return nil, err
	}
	var m MailLabelPayload
	if err := json.Unmarshal(r.body, &m); err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(r.body))
	}
	slog.Info("Received mail labels from ESI", "characterID", characterID, "count", len(m.Labels))
	return &m, err
}
