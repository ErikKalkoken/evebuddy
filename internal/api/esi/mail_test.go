package esi_test

import (
	"example/esiapp/internal/api/esi"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestFetchMail(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	fixture := `
		{
			"body": "blah blah blah",
			"from": 90000001,
			"labels": [
			2,
			32
			],
			"read": true,
			"subject": "test",
			"timestamp": "2015-09-30T16:07:00Z"
		}`

	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/latest/characters/1/mail/7/?token=token",
		httpmock.NewStringResponder(200, fixture),
	)

	c := http.Client{}
	// when
	m, err := esi.FetchMail(c, 1, 7, "token")

	// then
	if assert.NoError(t, err) {
		assert.Equal(t, int32(90000001), m.FromID)
		assert.Equal(t, "blah blah blah", m.Body)
		assert.Equal(t, "test", m.Subject)
		assert.Equal(t, []int32{2, 32}, m.Labels)
		assert.Equal(t, "2015-09-30T16:07:00Z", m.Timestamp)
	}
}
