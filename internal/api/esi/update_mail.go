package esi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type MailUpdate struct {
	Labels []int32 `json:"labels"`
	Read   bool    `json:"read"`
}

func UpdateMail(client *http.Client, characterID, mailID int32, content MailUpdate, tokenString string) (int32, error) {
	path := fmt.Sprintf("/characters/%d/mail/%d/", characterID, mailID)
	data, err := json.Marshal(content)
	if err != nil {
		return 0, err
	}
	_, err = raiseError(putESIWithToken(client, path, data, tokenString))
	if err != nil {
		return 0, err
	}
	slog.Info("Updated mail", "characterID", characterID, "mailID", mailID)
	return int32(mailID), err
}
