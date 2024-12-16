package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
)

type fakeToken struct {
	jwt.Token

	data    map[string]any
	subject string
}

func newFakeToken() fakeToken {
	f := fakeToken{
		data: make(map[string]any),
	}
	return f
}

func (t fakeToken) Get(k string) (any, bool) {
	x, ok := t.data[k]
	return x, ok
}

func (t fakeToken) Subject() string {
	return t.subject
}

type X = jwk.Set

type fakeJWKSet struct {
	X
}

var jwkSetData = map[string]any{
	"keys": []map[string]any{
		{
			"alg": "RS256",
			"e":   "AQAB",
			"kid": "JWT-Signature-Key",
			"kty": "RSA",
			"n":   "nehPQ7FQ1YK-leKyIg-aACZaT-DbTL5V1XpXghtLX_bEC-fwxhdE_4yQKDF6cA-V4c-5kh8wMZbfYw5xxgM9DynhMkVrmQFyYB3QMZwydr922UWs3kLz-nO6vi0ldCn-ffM9odUPRHv9UbhM5bB4SZtCrpr9hWQgJ3FjzWO2KosGQ8acLxLtDQfU_lq0OGzoj_oWwUKaN_OVfu80zGTH7mxVeGMJqWXABKd52ByvYZn3wL_hG60DfDWGV_xfLlHMt_WoKZmrXT4V3BCBmbitJ6lda3oNdNeHUh486iqaL43bMR2K4TzrspGMRUYXcudUQ9TycBQBrUlT85NRY9TeOw",
			"use": "sig",
		},
		{
			"alg": "ES256",
			"crv": "P-256",
			"kid": "8878a23f-2489-4045-989e-4d2f3ec1ae1a",
			"kty": "EC",
			"use": "sig",
			"x":   "PatzB2HJzZOzmqQyYpQYqn3SAXoVYWrZKmMgJnfK94I",
			"y":   "qDb1kUd13fRTN2UNmcgSoQoyqeF_C1MsFlY_a87csnY",
		},
	},
	"SkipUnresolvedJsonWebKeys": true,
}

func TestSSOEnd2End(t *testing.T) {
	ctx := context.Background()
	// monkey patching 3rd party packages
	openURLOrig := openURL
	openURL = func(u string) error {
		_, err := http.Get(u)
		return err
	}
	defer func() { openURL = openURLOrig }()
	jwkFetchOrig := jwkFetch
	jwkFetch = func(_ context.Context, _ string, _ ...jwk.FetchOption) (jwk.Set, error) {
		x := &fakeJWKSet{}
		return x, nil
	}
	defer func() { jwkFetch = jwkFetchOrig }()
	jwkParseStringOrig := jwkParseString
	jwkParseString = func(_ string, _ ...jwt.ParseOption) (jwt.Token, error) {
		t := newFakeToken()
		t.subject = "CHARACTER:EVE:1234567"
		t.data["name"] = "Bruce Wayne"
		return t, nil
	}
	defer func() { jwkParseString = jwkParseStringOrig }()
	authorizeHandler := func(r http.ResponseWriter, req *http.Request) {
		v1 := req.URL.Query()
		v2 := url.Values{}
		v2.Add("code", v1.Get("code"))
		v2.Add("state", v1.Get("state"))
		redirectURL := v1.Get("redirect_uri")
		http.Get(redirectURL + "?" + v2.Encode())
	}
	tokenHandler := func(w http.ResponseWriter, req *http.Request) {
		d := map[string]any{
			"access_token":  "access_token",
			"expires_in":    1199,
			"token_type":    "Bearer",
			"refresh_token": "refresh_token",
		}
		b, _ := json.Marshal(d)
		if _, err := w.Write(b); err != nil {
			t.Fatal(err)
		}
	}
	tokenHandlerWithError := func(w http.ResponseWriter, r *http.Request) {
		d := map[string]any{
			"error":             "error",
			"error_description": "error_description",
		}
		b, _ := json.Marshal(d)
		if _, err := w.Write(b); err != nil {
			t.Fatal(err)
		}
		w.Header()
	}
	jwksHandler := func(w http.ResponseWriter, req *http.Request) {
		b, _ := json.Marshal(jwkSetData)
		if _, err := w.Write(b); err != nil {
			t.Fatal(err)
		}
	}
	notFoundHandler := func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
		t.Fatal("unexpected URL: ", req.URL)
	}
	t.Run("can create new token", func(t *testing.T) {
		// given
		router := http.NewServeMux()
		router.HandleFunc("/authorize/", authorizeHandler)
		router.HandleFunc("/token/", tokenHandler)
		router.HandleFunc("/jwks/", jwksHandler)
		router.HandleFunc("/", notFoundHandler)
		server := httptest.NewServer(router)
		defer server.Close()
		s := new("client-id", http.DefaultClient, "/callback", 8000, server.URL+"/authorize", server.URL+"/token")
		// when
		token, err := s.Authenticate(ctx, []string{"alpha"})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "access_token", token.AccessToken)
			assert.Equal(t, "refresh_token", token.RefreshToken)
			assert.Equal(t, int32(1234567), token.CharacterID)
			assert.Equal(t, "Bruce Wayne", token.CharacterName)
			assert.WithinDuration(t, time.Now().UTC().Add(1199*time.Second), token.ExpiresAt, time.Second*5)
		}
	})
	t.Run("should return error when state is invalid for new token", func(t *testing.T) {
		// given
		router := http.NewServeMux()
		router.HandleFunc("/authorize/", func(r http.ResponseWriter, req *http.Request) {
			v1 := req.URL.Query()
			v2 := url.Values{}
			v2.Add("code", v1.Get("code"))
			v2.Add("state", "invalid")
			redirectURL := v1.Get("redirect_uri")
			http.Get(redirectURL + "?" + v2.Encode())
		})
		router.HandleFunc("/token/", tokenHandler)
		router.HandleFunc("/jwks/", jwksHandler)
		router.HandleFunc("/", notFoundHandler)
		server := httptest.NewServer(router)
		defer server.Close()
		s := new("client-id", http.DefaultClient, "/callback", 8000, server.URL+"/authorize", server.URL+"/token")
		// when
		_, err := s.Authenticate(ctx, []string{"alpha"})
		// then
		assert.Error(t, err)
	})
	t.Run("should return error when fetching token failed for new token", func(t *testing.T) {
		// given
		router := http.NewServeMux()
		router.HandleFunc("/authorize/", authorizeHandler)
		router.HandleFunc("/token/", tokenHandlerWithError)
		router.HandleFunc("/jwks/", jwksHandler)
		router.HandleFunc("/", notFoundHandler)
		server := httptest.NewServer(router)
		defer server.Close()
		s := new("client-id", http.DefaultClient, "/callback", 8000, server.URL+"/authorize", server.URL+"/token")
		// when
		_, err := s.Authenticate(ctx, []string{"alpha"})
		// then
		assert.Error(t, err)
	})
	t.Run("should return error when fetching jwks set failed for new token", func(t *testing.T) {
		// given
		router := http.NewServeMux()
		router.HandleFunc("/authorize/", authorizeHandler)
		router.HandleFunc("/token/", tokenHandler)
		router.HandleFunc("/jwks/", jwksHandler)
		router.HandleFunc("/", notFoundHandler)
		server := httptest.NewServer(router)
		defer server.Close()
		s := new("client-id", http.DefaultClient, "/callback", 8000, server.URL+"/authorize", server.URL+"/token")
		originalPatch := jwkFetch
		jwkFetch = func(_ context.Context, _ string, _ ...jwk.FetchOption) (jwk.Set, error) {
			return nil, fmt.Errorf("failed")
		}
		defer func() { jwkFetch = originalPatch }()
		// when
		_, err := s.Authenticate(ctx, []string{"alpha"})
		// then
		assert.Error(t, err)
	})
	t.Run("should return error when validating token failed for new token", func(t *testing.T) {
		// given
		router := http.NewServeMux()
		router.HandleFunc("/authorize/", authorizeHandler)
		router.HandleFunc("/token/", tokenHandler)
		router.HandleFunc("/jwks/", jwksHandler)
		router.HandleFunc("/", notFoundHandler)
		server := httptest.NewServer(router)
		defer server.Close()
		s := new("client-id", http.DefaultClient, "/callback", 8000, server.URL+"/authorize", server.URL+"/token")
		originalPatch := jwkParseString
		jwkParseString = func(s string, options ...jwt.ParseOption) (jwt.Token, error) {
			return nil, fmt.Errorf("failed")
		}
		defer func() { jwkParseString = originalPatch }()
		// when
		_, err := s.Authenticate(ctx, []string{"alpha"})
		// then
		assert.Error(t, err)
	})
	t.Run("can refresh token", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("/authorize/", authorizeHandler)
		router.HandleFunc("/token/", tokenHandler)
		router.HandleFunc("/jwks/", jwksHandler)
		router.HandleFunc("/", notFoundHandler)
		server := httptest.NewServer(router)
		defer server.Close()
		s := new("client-id", http.DefaultClient, "/callback", 8000, server.URL+"/authorize", server.URL+"/token")
		// when
		token, err := s.RefreshToken(ctx, "refreshToken")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "access_token", token.AccessToken)
			assert.Equal(t, "refresh_token", token.RefreshToken)
			assert.WithinDuration(t, time.Now().UTC().Add(1199*time.Second), token.ExpiresAt, time.Second*5)
		}
	})
	t.Run("should return error when no refresh token provided for refresh", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("/authorize/", authorizeHandler)
		router.HandleFunc("/token/", tokenHandler)
		router.HandleFunc("/jwks/", jwksHandler)
		router.HandleFunc("/", notFoundHandler)
		server := httptest.NewServer(router)
		defer server.Close()
		s := new("client-id", http.DefaultClient, "/callback", 8000, server.URL+"/authorize", server.URL+"/token")
		// when
		_, err := s.RefreshToken(ctx, "")
		// then
		assert.Error(t, err)
	})
	t.Run("should return error when fetching token failed for refresh", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("/authorize/", authorizeHandler)
		router.HandleFunc("/token/", tokenHandlerWithError)
		router.HandleFunc("/jwks/", jwksHandler)
		router.HandleFunc("/", notFoundHandler)
		server := httptest.NewServer(router)
		defer server.Close()
		s := new("client-id", http.DefaultClient, "/callback", 8000, server.URL+"/authorize", server.URL+"/token")
		// when
		_, err := s.RefreshToken(ctx, "refreshToken")
		// then
		assert.Error(t, err)
	})
	t.Run("should return error when validating token failed for refresh", func(t *testing.T) {
		router := http.NewServeMux()
		router.HandleFunc("/authorize/", authorizeHandler)
		router.HandleFunc("/token/", tokenHandler)
		router.HandleFunc("/jwks/", jwksHandler)
		router.HandleFunc("/", notFoundHandler)
		server := httptest.NewServer(router)
		defer server.Close()
		s := new("client-id", http.DefaultClient, "/callback", 8000, server.URL+"/authorize", server.URL+"/token")
		originalPatch := jwkParseString
		jwkParseString = func(s string, options ...jwt.ParseOption) (jwt.Token, error) {
			return nil, fmt.Errorf("failed")
		}
		defer func() { jwkParseString = originalPatch }()
		// when
		_, err := s.RefreshToken(ctx, "refreshToken")
		// then
		assert.Error(t, err)
	})
}

func TestSSO(t *testing.T) {
	t.Run("can create a new service with correct defaults", func(t *testing.T) {
		s := New("clientID", http.DefaultClient)
		assert.Equal(t, s.address(), "localhost:30123")
		assert.Equal(t, s.redirectURI(), "http://localhost:30123/callback")
	})
	t.Run("can generate a correct start URL", func(t *testing.T) {
		// given
		s := New("clientID", http.DefaultClient)
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
		s := New("abc", http.DefaultClient)
		s.tokenURL = server.URL
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
		s := New("abc", http.DefaultClient)
		s.tokenURL = server.URL
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
		s := New("abc", http.DefaultClient)
		s.tokenURL = server.URL
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
		s := New("abc", http.DefaultClient)
		s.tokenURL = server.URL
		// when
		_, err := s.fetchRefreshedToken("refreshToken")
		// then
		assert.ErrorIs(t, err, ErrTokenError)
	})
}
