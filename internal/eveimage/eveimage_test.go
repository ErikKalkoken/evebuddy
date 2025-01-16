package eveimage_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/cache"
	"github.com/ErikKalkoken/evebuddy/internal/eveimage"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestImageFetching(t *testing.T) {
	c := cache.New()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	dat, err := os.ReadFile("testdata/character_93330670_64.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("can fetch an alliance logo from the image server", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/alliances/99/logo?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := eveimage.New(c, http.DefaultClient, false)
		r, err := m.AllianceLogo(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a character portrait from the image server", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/characters/93330670/portrait?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := eveimage.New(c, http.DefaultClient, false)
		r, err := m.CharacterPortrait(93330670, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a corporation logo from the image server", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/corporations/99/logo?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := eveimage.New(c, http.DefaultClient, false)
		r, err := m.CorporationLogo(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a faction logo from the image server", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/corporations/99/logo?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := eveimage.New(c, http.DefaultClient, false)
		r, err := m.FactionLogo(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type icon from the image server", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/icon?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := eveimage.New(c, http.DefaultClient, false)
		r, err := m.InventoryTypeIcon(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type render from the image server", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/render?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := eveimage.New(c, http.DefaultClient, false)
		r, err := m.InventoryTypeRender(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type BPO from the image server", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/bp?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := eveimage.New(c, http.DefaultClient, false)
		r, err := m.InventoryTypeBPO(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type BPC from the image server", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/bpc?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := eveimage.New(c, http.DefaultClient, false)
		r, err := m.InventoryTypeBPC(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("should convert images size errors", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/characters/93330670/portrait?size=64",
			httpmock.NewBytesResponder(200, dat))
		m := eveimage.New(c, http.DefaultClient, false)
		// when
		_, err := m.CharacterPortrait(93330670, 0)
		// then
		assert.ErrorIs(t, err, eveimage.ErrInvalidSize)
	})
	t.Run("can clear cache", func(t *testing.T) {
		// given
		c.Clear()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/render?size=64",
			httpmock.NewBytesResponder(200, dat))
		m := eveimage.New(c, http.DefaultClient, false)
		_, err := m.InventoryTypeRender(99, 64)
		if err != nil {
			t.Fatal(err)
		}
		//when
		_, err = m.ClearCache()
		// then
		assert.NoError(t, err)
	})
}

func TestHTTPError(t *testing.T) {
	err := eveimage.HTTPError{StatusCode: 200, Status: "200 OK"}
	s := err.Error()
	assert.Equal(t, "HTTP error: 200 OK", s)
}
