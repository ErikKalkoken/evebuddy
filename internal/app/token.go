package app

import (
	"time"

	"github.com/ErikKalkoken/go-set"
)

// CharacterToken is a SSO token belonging to a character in Eve Online.
type CharacterToken struct {
	AccessToken  string
	CharacterID  int32
	ExpiresAt    time.Time
	ID           int64
	RefreshToken string
	Scopes       set.Set[string]
	TokenType    string
}

// RemainsValid reports whether a token remains valid within a duration.
func (ct CharacterToken) RemainsValid(d time.Duration) bool {
	return ct.ExpiresAt.After(time.Now().Add(d))
}

func (ct CharacterToken) HasScopes(scopes set.Set[string]) bool {
	return ct.Scopes.ContainsAll(scopes.All())
}

func (ct CharacterToken) MissingScopes(scopes set.Set[string]) bool {
	return ct.Scopes.ContainsAll(scopes.All())
}
