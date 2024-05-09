package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

const cacheTimeoutJWKSet = 6 * 3600

// Validate JWT token and return claims
func validateToken(tokenString string) (jwt.MapClaims, error) {
	// parse token and validate signature
	token, err := jwt.Parse(tokenString, getKey)
	if err != nil {
		return nil, err
	}

	// validate issuer claim
	claims := token.Claims.(jwt.MapClaims)
	iss := claims["iss"].(string)
	if iss != ssoIssuer1 && iss != ssoIssuer2 {
		return nil, fmt.Errorf("invalid issuer claim")
	}

	// validate audience claim
	aud := claims["aud"].([]any)
	if aud[0].(string) != ssoClientId {
		return nil, fmt.Errorf("invalid first audience claim")
	}
	if aud[1].(string) != "EVE Online" {
		return nil, fmt.Errorf("invalid 2nd audience claim")
	}

	return claims, nil
}

// getKey returns the public key for a JWT token.
func getKey(token *jwt.Token) (any, error) {
	set, err := fetchJWKSet()
	if err != nil {
		return nil, err
	}
	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have string kid")
	}

	key, ok := set.LookupKeyID(keyID)
	if !ok {
		return nil, fmt.Errorf("unable to find key %q", keyID)
	}

	var rawKey any
	if err := key.Raw(&rawKey); err != nil {
		return nil, fmt.Errorf("failed to create public key: %s", err)
	}
	return rawKey, nil
}

// fetchJWKSet returns the current JWK set from the web. It is cached.
func fetchJWKSet() (jwk.Set, error) {
	key := "jwk-set"
	v, found := cache.Get(key)
	if found {
		return v.(jwk.Set), nil
	}
	jwksURL, err := determineJwksURL()
	if err != nil {
		return nil, err
	}
	set, err := jwk.Fetch(context.Background(), jwksURL)
	if err != nil {
		return nil, err
	}
	cache.Set(key, set, cacheTimeoutJWKSet)
	return set, nil
}

// Determine URL for JWK sets dynamically from web site and return it
func determineJwksURL() (string, error) {
	resp, err := http.Get(oauthURL)
	if err != nil {
		return "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	jwksURL := data["jwks_uri"].(string)
	return jwksURL, nil
}
