package esi

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestSendRequest(t *testing.T) {
	c := &http.Client{}

	t.Run("should return body when successful", func(t *testing.T) {
		// given
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		fixture := `{"body": "blah blah blah"}`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/latest/dummy",
			httpmock.NewStringResponder(200, fixture),
		)
		// when
		b, err := getESI(c, "/dummy")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, []byte(fixture), b)
		}
	})
	t.Run("should return error on http error", func(t *testing.T) {
		// given
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		fixture := `{"error": "custom error"}`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/latest/dummy",
			httpmock.NewStringResponder(404, fixture),
		)
		// when
		b, err := getESI(c, "/dummy")
		// then
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "custom error")
			assert.Nil(t, b)
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
		}
	})
	t.Run("should return retry on 503 status", func(t *testing.T) {
		// given
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		fixture := `{"error": "custom error"}`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/latest/dummy",
			httpmock.NewStringResponder(503, fixture),
		)
		// when
		b, err := getESI(c, "/dummy")
		// then
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "custom error")
			assert.Nil(t, b)
			assert.Equal(t, 4, httpmock.GetTotalCallCount())
		}
	})
}
