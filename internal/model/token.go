package model

import "time"

// A SSO token belonging to a character in Eve Online.
type CharacterToken struct {
	AccessToken  string
	CharacterID  int32
	ExpiresAt    time.Time
	RefreshToken string
	Scopes       []string
	TokenType    string
}

// RemainsValid reports wether a token remains valid within a duration.
func (t *CharacterToken) RemainsValid(d time.Duration) bool {
	return t.ExpiresAt.After(time.Now().Add(d))
}
