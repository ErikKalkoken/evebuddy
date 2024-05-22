package images

import (
	"errors"
	"os"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestLoadResourceFromURL(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	dat, err := os.ReadFile("character_93330670_64.jpeg")
	if err != nil {
		panic(err)
	}
	url := "https://images.evetech.net/characters/93330670/portrait?size=64"
	t.Run("can fetch an image from the image server", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewBytesResponder(200, dat))
		//when
		r, err := loadResourceFromURL(url)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}

	})
	t.Run("should return error from http package", func(t *testing.T) {
		// given
		httpmock.Reset()
		errTest := errors.New("dummy error")
		httpmock.RegisterResponder("GET", url, httpmock.NewErrorResponder(errTest))
		//when
		_, err := loadResourceFromURL(url)
		// then
		assert.ErrorIs(t, err, errTest)
	})
	t.Run("should return error when an HTTP error occurred", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(400, ""))
		//when
		_, err := loadResourceFromURL(url)
		// then
		if assert.Error(t, err) {
			assert.IsType(t, HTTPError{}, err)
		}
	})
}
