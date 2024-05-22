package images_test

import (
	"os"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/images"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestImage(t *testing.T) {
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
		p := t.TempDir()
		m := images.New(p)
		r, err := m.CharacterPortrait(93330670, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("should return errors", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewBytesResponder(200, dat))
		p := t.TempDir()
		m := images.New(p)
		// when
		_, err := m.CharacterPortrait(93330670, 0)
		// then
		assert.ErrorIs(t, err, images.ErrInvalidSize)
	})
}
