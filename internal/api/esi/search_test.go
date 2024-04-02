package esi_test

import (
	"example/esiapp/internal/api/esi"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestSearch(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	fixture := `
		{
			"character": [1, 2, 3],
			"region": [4, 5, 6]
		}`

	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/latest/characters/1/search/",
		httpmock.NewStringResponder(200, fixture),
	)

	c := &http.Client{}
	// when
	categories := []esi.SearchCategory{esi.SearchCategoryCharacter}
	r, err := esi.Search(c, 1, "dummy", categories, "token")

	// then
	if assert.NoError(t, err) {
		assert.Equal(t, []int32{1, 2, 3}, r.Character)
		assert.Len(t, r.Corporation, 0)
		assert.Equal(t, []int32{4, 5, 6}, r.Region)
	}
}
