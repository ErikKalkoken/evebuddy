package sso

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	jwksURL     = "https://login.eveonline.com/oauth/jwks"
	ssoAudience = "EVE Online"
	ssoIssuer1  = "login.eveonline.com"
	ssoIssuer2  = "https://login.eveonline.com"
)

// validateJWT validates a JWT payload and when valid returns it as parsed object.
func validateJWT(ctx context.Context, accessToken string) (jwt.Token, error) {
	// fetch the JWK set
	set, err := jwk.Fetch(ctx, jwksURL)
	if err != nil {
		return nil, fmt.Errorf("fetching JWK set: %w", err)
	}
	// validate token
	token, err := jwt.ParseString(
		accessToken,
		jwt.WithKeySet(set),
		jwt.WithAudience(ssoAudience),
		jwt.WithValidator(jwt.ValidatorFunc(func(ctx context.Context, t jwt.Token) jwt.ValidationError {
			if x := t.Issuer(); x != ssoIssuer1 && x != ssoIssuer2 {
				return jwt.NewValidationError(fmt.Errorf("invalid issuer: %s", x))
			}
			return nil
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("parsing jwt: %w", err)
	}
	return token, err
}

// extractCharacterID returns the character ID in a JWT.
func extractCharacterID(token jwt.Token) (int, error) {
	p := strings.Split(token.Subject(), ":")
	if len(p) != 3 || p[0] != "CHARACTER" || p[1] != "EVE" {
		return 0, fmt.Errorf("invalid subject in JWK")
	}
	return strconv.Atoi(p[2])
}

// extractCharacterName returns the character name in a JWT.
func extractCharacterName(token jwt.Token) string {
	x, ok := token.Get("name")
	if !ok {
		return ""
	}
	return x.(string)
}

// extractScopes returns the scopes in a JWT.
func extractScopes(token jwt.Token) []string {
	scopes := make([]string, 0)
	x, ok := token.Get("scp")
	if !ok {
		return scopes
	}
	for _, s := range x.([]any) {
		scopes = append(scopes, s.(string))
	}
	return scopes
}
