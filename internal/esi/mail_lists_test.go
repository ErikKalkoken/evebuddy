package esi_test

import (
	"example/esiapp/internal/esi"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestFetchMailLists(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	fixture := `
	[
		{
		  "mailing_list_id": 1,
		  "name": "test_mailing_list"
		}
	  ]`

	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/latest/characters/1/mail/lists/?token=token",
		httpmock.NewStringResponder(200, fixture),
	)

	c := http.Client{}
	// when
	ll, err := esi.FetchMailLists(c, 1, "token")

	// then
	assert.Nil(t, err)
	assert.Len(t, ll, 1)
	o := ll[0]
	assert.Equal(t, int32(1), o.ID)
	assert.Equal(t, "test_mailing_list", o.Name)
}
