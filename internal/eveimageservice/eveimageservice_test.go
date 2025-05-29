package eveimageservice_test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

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
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Run("can fetch an alliance logo from the image server", func(t *testing.T) {
		// given
		c := newCache()
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
		c := newCache()
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
		c := newCache()
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
		c := newCache()
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
		c := newCache()
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
		c := newCache()
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
		c := newCache()
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
		c := newCache()
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
		c := newCache()
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
		assert.ErrorIs(t, err, eveimageservice.ErrInvalidSize)
	})
	t.Run("can clear cache", func(t *testing.T) {
		// given
		c := newCache()
		dat, err := os.ReadFile("testdata/character.jpeg")
		if err != nil {
			t.Fatal(err)
		}
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://images.evetech.net/types/99/render?size=64",
			httpmock.NewBytesResponder(200, dat),
		)
		m := eveimageservice.New(c, http.DefaultClient, false)
		if _, err := m.InventoryTypeRender(99, 64); err != nil {
			t.Fatal(err)
		}
		//when
		err = m.ClearCache()
		// then
		assert.NoError(t, err)
	})
}

func TestEntityIcon(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	c := newCache()
	m := eveimageservice.New(c, http.DefaultClient, false)
	alliance, err := os.ReadFile("testdata/alliance.png")
	if err != nil {
		t.Fatal(err)
	}
	httpmock.RegisterResponder(
		"GET",
		"https://images.evetech.net/alliances/42/logo?size=64",
		httpmock.NewBytesResponder(200, alliance),
	)
	character, err := os.ReadFile("testdata/character.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	httpmock.RegisterResponder(
		"GET",
		"https://images.evetech.net/characters/42/portrait?size=64",
		httpmock.NewBytesResponder(200, character),
	)
	corporation, err := os.ReadFile("testdata/corporation.png")
	if err != nil {
		t.Fatal(err)
	}
	httpmock.RegisterResponder(
		"GET",
		"https://images.evetech.net/corporations/42/logo?size=64",
		httpmock.NewBytesResponder(200, corporation),
	)
	faction, err := os.ReadFile("testdata/faction.png")
	if err != nil {
		t.Fatal(err)
	}
	httpmock.RegisterResponder(
		"GET",
		"https://images.evetech.net/corporations/888/logo?size=64",
		httpmock.NewBytesResponder(200, faction),
	)
	typ, err := os.ReadFile("testdata/type.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	httpmock.RegisterResponder(
		"GET",
		"https://images.evetech.net/types/42/icon?size=64",
		httpmock.NewBytesResponder(200, typ),
	)
	cases := []struct {
		id        int32
		category  string
		want      []byte
		wantError bool
	}{
		{42, "alliance", alliance, false},
		{42, "character", character, false},
		{42, "corporation", corporation, false},
		{888, "faction", faction, false},
		{42, "inventory_type", typ, false},
		{1, "invalid", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.category, func(t *testing.T) {
			c.Clear()
			r, err := m.EntityIcon(tc.id, tc.category, 64)
			if !tc.wantError {
				if assert.NoError(t, err) {
					got := r.Content()
					assert.Equal(t, tc.want, got)
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestHTTPError(t *testing.T) {
	err := eveimageservice.HTTPError{StatusCode: 200, Status: "200 OK"}
	s := err.Error()
	assert.Equal(t, "HTTP error: 200 OK", s)
}

func TestLive(t *testing.T) {
	s := eveimageservice.New(newCache(), http.DefaultClient, true)
	x, err := s.CharacterPortrait(123, 64)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, x.Content())
	}
}
