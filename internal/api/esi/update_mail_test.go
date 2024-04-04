package esi_test

import (
	"example/esiapp/internal/api/esi"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMail(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		"PUT",
		"https://esi.evetech.net/latest/characters/1/mail/2/",
		httpmock.NewStringResponder(204, ""),
	)
	c := &http.Client{}

	// when
	data := esi.MailUpdate{Read: true}
	_, err := esi.UpdateMail(c, 1, 2, data, "token")

	// then
	assert.NoError(t, err)
}
