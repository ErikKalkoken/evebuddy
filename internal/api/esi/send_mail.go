package esi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

type MailSend struct {
	Body       string          `json:"body"`
	Recipients []MailRecipient `json:"recipients"`
	Subject    string          `json:"subject"`
}

func SendMail(client *http.Client, characterID int32, tokenString string, mail MailSend) (int32, error) {
	path := fmt.Sprintf("/characters/%d/mail/", characterID)
	data, err := json.Marshal(mail)
	if err != nil {
		return 0, err
	}
	r, err := raiseError(postESIWithToken(client, path, data, tokenString))
	if err != nil {
		return 0, err
	}
	mailID, err := strconv.Atoi(string(r.body))
	if err != nil {
		return 0, fmt.Errorf("%v: %v", err, string(r.body))
	}
	slog.Info("Created new mail", "characterID", characterID, "mailID", mailID)
	return int32(mailID), err
}
