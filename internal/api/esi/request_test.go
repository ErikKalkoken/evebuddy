package esi

import (
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
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
		r, err := getESI(c, "/dummy")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, []byte(fixture), r.body)
		}
	})
	t.Run("should return esi error status as error", func(t *testing.T) {
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
		r, err := getESI(c, "/dummy")
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "404")
			assert.Contains(t, err.Error(), "custom error")
			assert.Nil(t, r)
		}
	})
	t.Run("should return esi error", func(t *testing.T) {
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
		r, err := getESIWithStatus(c, "/dummy")
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		if assert.NoError(t, err) {
			assert.False(t, r.ok())
			assert.Error(t, r.error())
			assert.Contains(t, r.error().Error(), "custom error")
		}
	})
	t.Run("should retry on 503 status", func(t *testing.T) {
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
		r, err := getESI(c, "/dummy")
		// then
		assert.Equal(t, 4, httpmock.GetTotalCallCount())
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "503")
			assert.Nil(t, r)
		}
	})
	t.Run("should return body from cache", func(t *testing.T) {
		// given
		cache.Clear()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		fixture := `{"body": "blah blah blah"}`
		h := make(http.Header)
		h.Add("Expires", time.Now().Add(time.Second*120).Format(time.RFC1123))
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/latest/dummy",
			httpmock.NewStringResponder(200, fixture).HeaderAdd(h),
		)
		_, err := getESI(c, "/dummy")
		if !assert.NoError(t, err) {
			t.Fail()
		}
		// when
		r, err := getESI(c, "/dummy")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, []byte(fixture), r.body)
		}
	})
}
