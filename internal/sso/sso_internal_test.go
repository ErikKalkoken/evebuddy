package sso

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSSO(t *testing.T) {
	t.Run("can create a new service", func(t *testing.T) {
		s := New("clientID", http.DefaultClient, nil)
		assert.Equal(t, s.address(), "localhost:30123")
		assert.Equal(t, s.redirectURI(), "http://localhost:30123/callback")
		assert.Equal(t, s.CallbackPath, callbackPathDefault)
		assert.Equal(t, s.Port, portDefault)
	})
	t.Run("can generate a correct start URL", func(t *testing.T) {
		// given
		s := New("clientID", http.DefaultClient, nil)
		// when
		got := s.makeStartURL("challenge", "state", []string{"esi-characters.read_blueprints.v1"})
		// then
		want := "https://login.eveonline.com/v2/oauth/authorize/?client_id=clientID&code_challenge=challenge&code_challenge_method=S256&redirect_uri=http%3A%2F%2Flocalhost%3A30123%2Fcallback&response_type=code&scope=esi-characters.read_blueprints.v1&state=state"
		u1, _ := url.Parse(got)
		u2, _ := url.Parse(want)
		assert.Equal(t, u2.Host, u1.Host)
		assert.Equal(t, u2.Path, u1.Path)
		q1, _ := url.ParseQuery(u1.RawQuery)
		q2, _ := url.ParseQuery(u2.RawQuery)
		assert.Equal(t, q2, q1)
	})
	t.Run("can generate code challenge", func(t *testing.T) {
		got, _ := calcCodeChallenge("abc")
		want := "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0"
		assert.Equal(t, want, got)
	})
	t.Run("can generate random string", func(t *testing.T) {
		s1, err := generateRandomStringBase64(16)
		if assert.NoError(t, err) {
			assert.Greater(t, len(s1), 0)
			s2, err := generateRandomStringBase64(16)
			if assert.NoError(t, err) {
				assert.Greater(t, len(s2), 0)
			}
			assert.NotEqual(t, s2, s1)
		}
	})
}

func TestSSOFetchNewToken(t *testing.T) {
	t.Run("can fetch new token from API", func(t *testing.T) {
		// given
		var actualRequestBody []byte
		var actualRequestHeader http.Header
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			var err error
			actualRequestBody, err = io.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			d := map[string]any{
				"access_token":  "access_token",
				"expires_in":    1199,
				"token_type":    "Bearer",
				"refresh_token": "refresh_token",
			}
			b, _ := json.Marshal(d)
			if _, err := rw.Write(b); err != nil {
				t.Fatal(err)
			}
			actualRequestHeader = req.Header.Clone()
		}))
		defer server.Close()
		s := New("abc", http.DefaultClient, nil)
		s.SSOTokenURL = server.URL
		// when
		x, err := s.fetchNewToken("code", "codeVerifier")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "application/x-www-form-urlencoded", actualRequestHeader.Get("Content-Type"))
			v, err := url.ParseQuery(string(actualRequestBody))
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "abc", v.Get("client_id"))
			assert.Equal(t, "authorization_code", v.Get("grant_type"))
			assert.Equal(t, "code", v.Get("code"))
			assert.Equal(t, "codeVerifier", v.Get("code_verifier"))

			assert.Equal(t, "access_token", x.AccessToken)
			assert.Equal(t, 1199, x.ExpiresIn)
			assert.Equal(t, "Bearer", x.TokenType)
			assert.Equal(t, "refresh_token", x.RefreshToken)
		}
	})
	t.Run("should return error when API returns error in payload", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			d := map[string]any{
				"error":             "error",
				"error_description": "error_description",
			}
			b, _ := json.Marshal(d)
			if _, err := rw.Write(b); err != nil {
				t.Fatal(err)
			}
		}))
		defer server.Close()
		s := New("abc", http.DefaultClient, nil)
		s.SSOTokenURL = server.URL
		// when
		_, err := s.fetchNewToken("code", "codeVerifier")
		// then
		assert.ErrorIs(t, err, ErrTokenError)
	})
}

func TestSSOFetchRefreshedToken(t *testing.T) {
	t.Run("can retrieve refreshed token from SSO", func(t *testing.T) {
		// given
		var actualRequestBody []byte
		var actualRequestHeader http.Header
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			var err error
			actualRequestBody, err = io.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			d := map[string]any{
				"access_token":  "access_token",
				"expires_in":    1199,
				"token_type":    "Bearer",
				"refresh_token": "refresh_token",
			}
			b, _ := json.Marshal(d)
			if _, err := rw.Write(b); err != nil {
				t.Fatal(err)
			}
			actualRequestHeader = req.Header.Clone()
		}))
		defer server.Close()
		s := New("abc", http.DefaultClient, nil)
		s.SSOTokenURL = server.URL
		// when
		x, err := s.fetchRefreshedToken("refreshToken")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "application/x-www-form-urlencoded", actualRequestHeader.Get("Content-Type"))
			v, err := url.ParseQuery(string(actualRequestBody))
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "abc", v.Get("client_id"))
			assert.Equal(t, "refresh_token", v.Get("grant_type"))
			assert.Equal(t, "refreshToken", v.Get("refresh_token"))

			assert.Equal(t, "access_token", x.AccessToken)
			assert.Equal(t, 1199, x.ExpiresIn)
			assert.Equal(t, "Bearer", x.TokenType)
			assert.Equal(t, "refresh_token", x.RefreshToken)
		}
	})
	t.Run("should return error when API returns error in payload", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			d := map[string]any{
				"error":             "error",
				"error_description": "error_description",
			}
			b, _ := json.Marshal(d)
			if _, err := rw.Write(b); err != nil {
				t.Fatal(err)
			}
		}))
		defer server.Close()
		s := New("abc", http.DefaultClient, nil)
		s.SSOTokenURL = server.URL
		// when
		_, err := s.fetchRefreshedToken("refreshToken")
		// then
		assert.ErrorIs(t, err, ErrTokenError)
	})
}

func TestDetermineJWKsURL(t *testing.T) {
	t.Run("can retrieve url", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			var err error
			_, err = io.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			d := map[string]any{
				"jwks_uri": "https://login.eveonline.com/oauth/jwks",
			}
			b, _ := json.Marshal(d)
			if _, err := rw.Write(b); err != nil {
				t.Fatal(err)
			}
		}))
		defer server.Close()
		s := New("abc", http.DefaultClient, nil)
		s.SSOTokenURL = server.URL
		// when
		x, err := s.determineJWKsURL()
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "https://login.eveonline.com/oauth/jwks", x)
		}
	})
}
