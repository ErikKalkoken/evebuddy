package app

import (
	"slices"
	"time"

	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"
)

// CharacterToken is a SSO token belonging to a character in Eve Online.
type CharacterToken struct {
	AccessToken  string
	CharacterID  int64
	ExpiresAt    time.Time
	ID           int64
	RefreshToken string
	Scopes       set.Set[string]
	TokenType    string
}

// RemainsValid reports whether a token remains valid within a duration.
func (t CharacterToken) RemainsValid(d time.Duration) bool {
	return t.ExpiresAt.After(time.Now().Add(d))
}

func (t CharacterToken) HasScopes(scopes set.Set[string]) bool {
	return t.Scopes.ContainsAll(scopes.All())
}

func (t CharacterToken) MissingScopes(scopes set.Set[string]) bool {
	return t.Scopes.ContainsAll(scopes.All())
}

func (t CharacterToken) AuthToken() *eveauth.Token {
	token2 := &eveauth.Token{
		AccessToken:  t.AccessToken,
		CharacterID:  int32(t.CharacterID),
		ExpiresAt:    t.ExpiresAt,
		RefreshToken: t.RefreshToken,
		Scopes:       slices.Collect(t.Scopes.All()),
		TokenType:    t.TokenType,
	}
	return token2
}
