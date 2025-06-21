package app

import (
	"errors"
	"time"
)

var (
	ErrTokenError = errors.New("token error")
)

// Token represents an OAuth token for a character in Eve Online.
type Token struct {
	AccessToken   string
	CharacterID   int32
	CharacterName string
	ExpiresAt     time.Time
	RefreshToken  string
	Scopes        []string
	TokenType     string
}
