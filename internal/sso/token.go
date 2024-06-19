package sso

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OAuth Token for a character in Eve Online.
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
	fmt.Println(claims)
	// calc character ID
	sub, err := claims.GetSubject()
	if err != nil {
		return nil, err
	}
	characterID, err := strconv.Atoi(strings.Split(sub, ":")[2])
	if err != nil {
		return nil, err
	}
	scopes := claims["scp"].([]any)
	t := &Token{
		AccessToken:   rawToken.AccessToken,
		CharacterID:   int32(characterID),
		CharacterName: claims["name"].(string),
		ExpiresAt:     rawToken.expiresAt(),
		RefreshToken:  rawToken.RefreshToken,
		TokenType:     rawToken.TokenType,
		Scopes:        make([]string, len(scopes)),
	}
	// Add scopes
	for i, v := range scopes {
		t.Scopes[i] = v.(string)
	}
	return t, nil
}

// token payload as returned from SSO API
type tokenPayload struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int32  `json:"expires_in"`
	TokenType        string `json:"token_type"`
	RefreshToken     string `json:"refresh_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// expiresAt returns the time when this token will expire.
func (t *tokenPayload) expiresAt() time.Time {
	x := time.Now().Add(time.Second * time.Duration(t.ExpiresIn))
	return x
}
