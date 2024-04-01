package esi_test

import (
	"example/esiapp/internal/api/esi"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestResolveEntityIDs(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	c := &http.Client{}

	t.Run("should return data", func(t *testing.T) {
		json := `
		[
			{
				"category": "character",
				"id": 95465499,
				"name": "CCP Bartender"
			},
			{
				"category": "solar_system",
				"id": 30000142,
				"name": "Jita"
			}
		]`
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/latest/universe/names/",
			httpmock.NewStringResponder(200, json),
		)

		// when
		r, err := esi.ResolveEntityIDs(c, []int32{95465499, 30000142})

		// then
		assert.Nil(t, err)
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.Len(t, r, 2)
		assert.Equal(t, int32(95465499), r[0].ID)
		assert.Equal(t, "CCP Bartender", r[0].Name)
		assert.Equal(t, "character", r[0].Category)
	})
	t.Run("should return error when too many IDs", func(t *testing.T) {
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/latest/universe/names/",
			httpmock.NewStringResponder(200, ""),
		)
		ids := make([]int32, 1100)
		for i := range int32(1100) {
			ids[i] = i
		}
		// when
		r, err := esi.ResolveEntityIDs(c, ids)
		// then
		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestResolveEntityNames(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	c := &http.Client{}

	t.Run("should return data", func(t *testing.T) {
		json := `
		{
			"characters": [
			  {
				"id": 95465499,
				"name": "CCP Bartender"
			  },
			  {
				"id": 2112625428,
				"name": "CCP Zoetrope"
			  }
			],
			"systems": [
			  {
				"id": 30000142,
				"name": "Jita"
			  }
			]
		}`
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/latest/universe/ids/",
			httpmock.NewStringResponder(200, json),
		)

		// when
		r, err := esi.ResolveEntityNames(c, []string{"alpha", "bravo"})

		// then
		assert.Nil(t, err)
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.Equal(t, int32(95465499), r.Characters[0].ID)
		assert.Equal(t, "CCP Bartender", r.Characters[0].Name)
	})
	t.Run("should return error when too many IDs", func(t *testing.T) {
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/latest/universe/ids/",
			httpmock.NewStringResponder(200, ""),
		)
		names := make([]string, 1100)
		for i := range 1100 {
			names[i] = "dummy"
		}
		// when
		r, err := esi.ResolveEntityNames(c, names)
		// then
		assert.Error(t, err)
		assert.Nil(t, r)
	})
}
