package eveimage

import (
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestLoadResourceFromURL(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	dat, err := os.ReadFile("testdata/character_93330670_64.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	url := "https://images.evetech.net/characters/93330670/portrait?size=64"
	t.Run("can fetch an image from the image server", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewBytesResponder(200, dat))
		//when
		x, err := loadDataFromURL(url, http.DefaultClient)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, x)
		}

	})
	t.Run("should return error from http package", func(t *testing.T) {
		// given
		httpmock.Reset()
		errTest := errors.New("dummy error")
		httpmock.RegisterResponder("GET", url, httpmock.NewErrorResponder(errTest))
		//when
		_, err := loadDataFromURL(url, http.DefaultClient)
		// then
		assert.ErrorIs(t, err, errTest)
	})
	t.Run("should return error when an HTTP error occurred", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(400, ""))
		//when
		_, err := loadDataFromURL(url, http.DefaultClient)
		// then
		if assert.Error(t, err) {
			assert.IsType(t, HTTPError{}, err)
		}
	})
}

type cache map[string][]byte

func newCache() cache {
	return make(cache)
}

func (c cache) Get(k string) ([]byte, bool) {
	v, ok := c[k]
	return v, ok
}

func (c cache) Set(k string, v []byte, d time.Duration) {
	c[k] = v
}

func (c cache) Clear() {
	for k := range c {
		delete(c, k)
	}
}

func TestImageFetching(t *testing.T) {
	c := newCache()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	dat, err := os.ReadFile("testdata/character_93330670_64.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	url := "https://images.evetech.net/alliances/99/logo?size=64"
	t.Run("can fetch image from the image server", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewBytesResponder(200, dat))
		//when
		m := New(c, http.DefaultClient, false)
		r, err := m.image(url, 0)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("should return dummy image when offline", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewBytesResponder(200, dat))
		//when
		m := New(c, http.DefaultClient, true)
		r, err := m.image(url, 0)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, resourceQuestionmark32Png, r)
		}
	})
	t.Run("can fetch a SKIN type from the image server", func(t *testing.T) {
		// given
		c.Clear()
		// when
		m := New(c, http.DefaultClient, false)
		r, err := m.InventoryTypeSKIN(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, resourceSkinicon64pxPng, r)
		}
	})
}
