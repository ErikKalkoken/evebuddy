package eveimageservice_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
)

func TestImageFetching(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Run("can fetch an alliance logo from the image server", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		dat, err := os.ReadFile("testdata/alliance.png")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/alliances/99/logo?size=64",
			httpmock.NewBytesResponder(200, dat))
		//when
		m := eveimageservice.New(c, http.DefaultClient, false)
		r, err := m.AllianceLogo(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a character portrait from the image server", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		dat, err := os.ReadFile("testdata/character.jpeg")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/characters/93330670/portrait?size=64",
			httpmock.NewBytesResponder(200, dat),
		)
		//when
		m := eveimageservice.New(c, http.DefaultClient, false)
		r, err := m.CharacterPortrait(93330670, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a corporation logo from the image server", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		dat, err := os.ReadFile("testdata/corporation.png")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/corporations/99/logo?size=64",
			httpmock.NewBytesResponder(200, dat),
		)
		//when
		m := eveimageservice.New(c, http.DefaultClient, false)
		r, err := m.CorporationLogo(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a faction logo from the image server", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		dat, err := os.ReadFile("testdata/faction.png")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/corporations/99/logo?size=64",
			httpmock.NewBytesResponder(200, dat),
		)
		//when
		m := eveimageservice.New(c, http.DefaultClient, false)
		r, err := m.FactionLogo(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type icon from the image server", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		dat, err := os.ReadFile("testdata/type.jpeg")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/icon?size=64",
			httpmock.NewBytesResponder(200, dat),
		)
		//when
		m := eveimageservice.New(c, http.DefaultClient, false)
		r, err := m.InventoryTypeIcon(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type render from the image server", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		dat, err := os.ReadFile("testdata/type.jpeg")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/render?size=64",
			httpmock.NewBytesResponder(200, dat),
		)
		//when
		m := eveimageservice.New(c, http.DefaultClient, false)
		r, err := m.InventoryTypeRender(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type BPO from the image server", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		dat, err := os.ReadFile("testdata/type.jpeg")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/bp?size=64",
			httpmock.NewBytesResponder(200, dat),
		)
		//when
		m := eveimageservice.New(c, http.DefaultClient, false)
		r, err := m.InventoryTypeBPO(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("can fetch a type BPC from the image server", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		dat, err := os.ReadFile("testdata/type.jpeg")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/bpc?size=64",
			httpmock.NewBytesResponder(200, dat),
		)
		//when
		m := eveimageservice.New(c, http.DefaultClient, false)
		r, err := m.InventoryTypeBPC(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, dat, r.Content())
		}
	})
	t.Run("should convert images size errors", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		dat, err := os.ReadFile("testdata/character.jpeg")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/characters/93330670/portrait?size=64",
			httpmock.NewBytesResponder(200, dat),
		)
		m := eveimageservice.New(c, http.DefaultClient, false)
		// when
		_, err = m.CharacterPortrait(93330670, 0)
		// then
		assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
	})
	t.Run("should return placeholder and not access network in offline mode", func(t *testing.T) {
		// given
		c := testutil.NewCacheFake()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/alliances/99/logo?size=64",
			httpmock.NewStringResponder(200, ""))
		//when
		m := eveimageservice.New(c, http.DefaultClient, true)
		r, err := m.AllianceLogo(99, 64)
		// then
		if assert.NoError(t, err) {
			assert.Contains(t, r.Name(), "question")
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
		}
	})
}

func TestHTTPError(t *testing.T) {
	err := eveimageservice.HTTPError{StatusCode: 200, Status: "200 OK"}
	s := err.Error()
	assert.Equal(t, "HTTP error: 200 OK", s)
}

func TestOffline(t *testing.T) {
	c := testutil.NewCacheFake()
	s := eveimageservice.New(c, http.DefaultClient, true)
	x, err := s.CharacterPortrait(123, 64)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, x.Content())
	}
}
