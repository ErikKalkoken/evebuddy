package esi_test

import (
	"example/esiapp/internal/api/esi"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestSendMail(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		"POST",
		"https://esi.evetech.net/latest/characters/1/mail/",
		httpmock.NewStringResponder(200, "42"),
	)
	c := &http.Client{}

	// when
	mail := esi.MailSend{Body: "body", Subject: "subject", Recipients: []esi.MailRecipient{{ID: 7, Type: "character"}}}
	r, err := esi.SendMail(c, 1, "token", mail)

	// then
	if assert.NoError(t, err) {
		assert.Equal(t, int32(42), r)

	}
}
