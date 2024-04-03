package logic

import (
	"example/esiapp/internal/api/esi"
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: time.Second * 30, // Timeout after 30 seconds
}

func SendMail(characterID int32, subject string, recipients []esi.MailRecipient, body string) error {
	token, err := FetchValidToken(characterID)
	if err != nil {
		return err
	}
	m := esi.MailSend{
		Body:       body,
		Subject:    subject,
		Recipients: recipients,
	}
	_, err = esi.SendMail(httpClient, characterID, token.AccessToken, m)
	if err != nil {
		return err
	}
	return nil
}
