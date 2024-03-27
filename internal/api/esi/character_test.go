package esi_test

import (
	"example/esiapp/internal/api/esi"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCanFetchCharacter(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	json := `
		{
			"birthday": "2015-03-24T11:37:00Z",
			"bloodline_id": 3,
			"corporation_id": 109299958,
			"description": "",
			"gender": "male",
			"name": "CCP Bartender",
			"race_id": 2,
			"title": "All round pretty awesome guy"
		}`

	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/latest/characters/1/",
		httpmock.NewStringResponder(200, json),
	)

	c := http.Client{}

	// when
	r, err := esi.FetchCharacter(c, 1)

	// then
	assert.Nil(t, err)
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
	assert.Equal(t, "CCP Bartender", r.Name)
}
