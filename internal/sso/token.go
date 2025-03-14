package sso

import (
	"time"
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
func newToken(rawToken *tokenPayload, characterID int, characterName string, scopes []string) *Token {
	t := &Token{
		AccessToken:   rawToken.AccessToken,
		CharacterID:   int32(characterID),
		CharacterName: characterName,
		ExpiresAt:     rawToken.expiresAt(),
		RefreshToken:  rawToken.RefreshToken,
		TokenType:     rawToken.TokenType,
		Scopes:        scopes,
	}
	return t
}

// token payload as returned from SSO API
type tokenPayload struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
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
