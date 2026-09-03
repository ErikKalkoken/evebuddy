package xgoesi_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fnt-eve/goesi-openapi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type tokenSourceFake struct {
	token       []*oauth2.Token
	errNotFound error
}

func (ts *tokenSourceFake) Token() (*oauth2.Token, error) {
	token, ok := xslices.Pop(&ts.token)
	if !ok {
		return nil, ts.errNotFound
	}
	return token, nil
}

var _ oauth2.TokenSource = (*tokenSourceFake)(nil)

func TestTokenTransport(t *testing.T) {
	const (
		characterID        = 42
		accessTokenExpired = "tokenExpired"
		accessTokenValid   = "tokenValid"
	)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	implants := []int64{1234}
	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("https://esi.evetech.net/characters/%d/implants", characterID),
		func(req *http.Request) (*http.Response, error) {
			token, err := extractBearerToken(req)
			if err != nil || token != accessTokenValid {
				resp := httpmock.NewJsonResponseOrPanic(http.StatusUnauthorized, map[string]any{
					"error": "unauthorized",
				})
				return resp, nil
			}
			resp := httpmock.NewJsonResponseOrPanic(http.StatusOK, implants)
			return resp, nil
		},
	)

	httpClient := &http.Client{
		Transport: &xgoesi.TokenRefresher{},
	}
	client := goesi.NewESIClientWithOptions(httpClient, goesi.ClientOptions{
		UserAgent: "MyApp/1.0 (contact@example.com)",
	})

	tokenExpired := &oauth2.Token{
		AccessToken:  accessTokenExpired,
		TokenType:    "Bearer",
		RefreshToken: "refreshToken1",
		Expiry:       time.Now().Add(-1 * time.Minute),
		ExpiresIn:    0,
	}
	tokenValid := &oauth2.Token{
		AccessToken:  accessTokenValid,
		TokenType:    "Bearer",
		RefreshToken: "refreshToken2",
		Expiry:       time.Now().Add(10 * time.Minute),
		ExpiresIn:    600,
	}
	// TokenType is empty here to mirror this app's real oauth2.TokenSource,
	// which never sets it (see internal/app/token.go).
	tokenValidNoType := &oauth2.Token{
		AccessToken:  accessTokenValid,
		RefreshToken: "refreshToken3",
		Expiry:       time.Now().Add(10 * time.Minute),
		ExpiresIn:    600,
	}

	t.Run("should work normally when token is valid", func(t *testing.T) {
		// given
		ts := &tokenSourceFake{token: []*oauth2.Token{tokenValid}}
		ctx := xgoesi.NewContextWithAuth(t.Context(), characterID, ts)

		// when
		got, resp, err := client.ClonesAPI.GetCharactersCharacterIdImplants(ctx, characterID).Execute()

		// then
		require.NoError(t, err)
		xassert.Equal(t, http.StatusOK, resp.StatusCode)
		xassert.Equal(t, implants, got)
	})

	t.Run("should refresh token", func(t *testing.T) {
		// given
		ts := &tokenSourceFake{token: []*oauth2.Token{tokenValid, tokenExpired}}
		ctx := xgoesi.NewContextWithAuth(t.Context(), characterID, ts)

		// when
		got, resp, err := client.ClonesAPI.GetCharactersCharacterIdImplants(ctx, characterID).Execute()

		// then
		require.NoError(t, err)
		xassert.Equal(t, http.StatusOK, resp.StatusCode)
		xassert.Equal(t, implants, got)
	})

	t.Run("should refresh token when refreshed token has no token type", func(t *testing.T) {
		// given
		ts := &tokenSourceFake{token: []*oauth2.Token{tokenValidNoType, tokenExpired}}
		ctx := xgoesi.NewContextWithAuth(t.Context(), characterID, ts)

		// when
		got, resp, err := client.ClonesAPI.GetCharactersCharacterIdImplants(ctx, characterID).Execute()

		// then
		require.NoError(t, err)
		xassert.Equal(t, http.StatusOK, resp.StatusCode)
		xassert.Equal(t, implants, got)
	})

	t.Run("should report error when token can not be refreshed", func(t *testing.T) {
		// given
		errNotFound := fmt.Errorf("no token found")
		ts := &tokenSourceFake{token: []*oauth2.Token{tokenExpired}, errNotFound: errNotFound}
		ctx := xgoesi.NewContextWithAuth(t.Context(), characterID, ts)

		// when
		_, _, err := client.ClonesAPI.GetCharactersCharacterIdImplants(ctx, characterID).Execute()

		// then
		assert.ErrorIs(t, err, errNotFound)
	})

	t.Run("should report 401 when no token provided", func(t *testing.T) {
		// given

		// when
		_, resp, err := client.ClonesAPI.GetCharactersCharacterIdImplants(t.Context(), characterID).Execute()

		// then
		require.Error(t, err)
		xassert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header missing")
	}

	// Option A: Using strings.Cut (Go 1.18+)
	prefix, token, found := strings.Cut(authHeader, " ")
	if !found || !strings.EqualFold(prefix, "Bearer") {
		return "", fmt.Errorf("invalid authorization header format")
	}
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	return token, nil
}
