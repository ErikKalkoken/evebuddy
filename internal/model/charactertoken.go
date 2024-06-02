package model

import "time"

// A SSO token belonging to a character in Eve Online.
type CharacterToken struct {
	AccessToken  string
	CharacterID  int32
	ExpiresAt    time.Time
	ID           int64
	RefreshToken string
	Scopes       []string
	TokenType    string
}

// RemainsValid reports wether a token remains valid within a duration.
func (ct CharacterToken) RemainsValid(d time.Duration) bool {
	return ct.ExpiresAt.After(time.Now().Add(d))
}
