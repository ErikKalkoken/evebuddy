package images_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/images"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestImageFetching(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	dat, err := os.ReadFile("character_93330670_64.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("can fetch an alliance logo from the image server", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/alliances/99/logo?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := images.New(t.TempDir(), http.DefaultClient)
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
		m := images.New(t.TempDir(), http.DefaultClient)
		r, err := m.CharacterPortrait(93330670, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a corporation logo from the image server", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/corporations/99/logo?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := images.New(t.TempDir(), http.DefaultClient)
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
		m := images.New(t.TempDir(), http.DefaultClient)
		r, err := m.FactionLogo(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type icon from the image server", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/icon?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := images.New(t.TempDir(), http.DefaultClient)
		r, err := m.InventoryTypeIcon(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type render from the image server", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/render?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := images.New(t.TempDir(), http.DefaultClient)
		r, err := m.InventoryTypeRender(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("should return errors", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/characters/93330670/portrait?size=64",
			httpmock.NewBytesResponder(200, dat))
		m := images.New(t.TempDir(), http.DefaultClient)
		// when
		_, err := m.CharacterPortrait(93330670, 0)
		// then
		assert.ErrorIs(t, err, images.ErrInvalidSize)
	})
	t.Run("can clear cache", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/render?size=64",
			httpmock.NewBytesResponder(200, dat))
		m := images.New(t.TempDir(), http.DefaultClient)
		_, err := m.InventoryTypeRender(99, 64)
		if err != nil {
			t.Fatal(err)
		}
		c, err := m.Count()
		if err != nil {
			t.Fatal(err)
		}
		if c != 1 {
			t.Fatal("unexpected count")
		}
		//when
		c, err = m.Clear()
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 1, c)
			c, err := m.Count()
			if assert.NoError(t, err) {
				assert.Equal(t, 0, c)
			}
		}
	})
	t.Run("can return file count", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/render?size=64",
			httpmock.NewBytesResponder(200, dat))
		m := images.New(t.TempDir(), http.DefaultClient)
		_, err := m.InventoryTypeRender(99, 64)
		if err != nil {
			t.Fatal(err)
		}
		//when
		c, err := m.Count()
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 1, c)
		}
	})
	t.Run("can return cache size", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/render?size=64",
			httpmock.NewBytesResponder(200, dat))
		m := images.New(t.TempDir(), http.DefaultClient)
		_, err := m.InventoryTypeRender(99, 64)
		if err != nil {
			t.Fatal(err)
		}
		//when
		x, err := m.Size()
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 4821, x)
		}
	})
}

func TestHTTPError(t *testing.T) {
	err := images.HTTPError{StatusCode: 200, Status: "200 OK"}
	s := err.Error()
	assert.Equal(t, "HTTP error: 200 OK", s)
}
