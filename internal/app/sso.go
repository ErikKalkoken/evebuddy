package app

import (
	"errors"
	"time"
)

var (
	ErrTokenError = errors.New("token error")
)

// Token represents an OAuth token for a character in EVE Online.
type Token struct {
	AccessToken   string
	CharacterID   int64
	CharacterName string
	ExpiresAt     time.Time
	RefreshToken  string
	Scopes        []string
	TokenType     string
}
