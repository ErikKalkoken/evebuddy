package sso

import (
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SSO token for Eve Online
type Token struct {
	AccessToken   string
	CharacterID   int32
	CharacterName string
	ExpiresAt     time.Time
	RefreshToken  string
	Scopes        []string
	TokenType     string
}

// newToken creates e new token and returns it.
func newToken(rawToken *tokenPayload, claims jwt.MapClaims) (*Token, error) {
	// calc character ID
	characterID, err := strconv.Atoi(strings.Split(claims["sub"].(string), ":")[2])
	if err != nil {
		return nil, err
	}
	scopes := claims["scp"].([]interface{})
	t := Token{
		AccessToken:   rawToken.AccessToken,
		CharacterID:   int32(characterID),
		CharacterName: claims["name"].(string),
		ExpiresAt:     calcExpiresAt(rawToken),
		RefreshToken:  rawToken.RefreshToken,
		TokenType:     rawToken.TokenType,
		Scopes:        make([]string, len(scopes)),
	}
	// Add scopes
	for i, v := range scopes {
		t.Scopes[i] = v.(string)
	}
	return &t, nil
}

func calcExpiresAt(rawToken *tokenPayload) time.Time {
	expiresAt := time.Now().Add(time.Second * time.Duration(rawToken.ExpiresIn))
	return expiresAt
}
