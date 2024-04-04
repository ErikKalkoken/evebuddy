package esi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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

// DeleteMail deletes a mail via ESI.
func DeleteMail(client *http.Client, characterID int32, mailID int32, tokenString string) error {
	path := fmt.Sprintf("/characters/%d/mail/%d/", characterID, mailID)
	_, err := raiseError(deleteESIWithToken(client, path, tokenString))
	if err != nil {
		return err
	}
	slog.Info("Deleted mail via ESI", "characterID", characterID, "mailID", mailID)
	return nil
}

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

type MailUpdate struct {
	Labels []int32 `json:"labels"`
	Read   bool    `json:"read"`
}

func UpdateMail(client *http.Client, characterID, mailID int32, content MailUpdate, tokenString string) error {
	path := fmt.Sprintf("/characters/%d/mail/%d/", characterID, mailID)
	data, err := json.Marshal(content)
	if err != nil {
		return err
	}
	_, err = raiseError(putESIWithToken(client, path, data, tokenString))
	if err != nil {
		return err
	}
	slog.Info("Updated mail", "characterID", characterID, "mailID", mailID)
	return err
}
