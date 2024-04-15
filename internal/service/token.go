package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/antihax/goesi"

	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/repository"
)

// A SSO token belonging to a character.
type Token struct {
	AccessToken  string
	CharacterID  int32
	ExpiresAt    time.Time
	RefreshToken string
	TokenType    string
}

func tokenFromDBModel(t repository.Token) Token {
	if t.CharacterID == 0 {
		panic("missing character ID")
	}
	return Token{
		AccessToken:  t.AccessToken,
		CharacterID:  int32(t.CharacterID),
		ExpiresAt:    t.ExpiresAt,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
	}
}

// GetValidToken returns a valid token for a character. Convenience function.
func (s *Service) GetValidToken(characterID int32) (*Token, error) {
	t, err := s.q.GetToken(context.Background(), int64(characterID))
	if err != nil {
		return nil, err
	}
	t2 := tokenFromDBModel(t)
	if err := s.EnsureValid(&t2); err != nil {
		return nil, err
	}
	return &t2, nil
}

// RemainsValid reports wether a token remains valid within a duration
func (t *Token) RemainsValid(d time.Duration) bool {
	return t.ExpiresAt.After(time.Now().Add(d))
}

func (s *Service) UpdateOrCreateToken(t *Token) error {
	if t.CharacterID == 0 {
		return errors.New("can not save token without character")
	}
	ctx := context.Background()
	arg := repository.CreateTokenParams{
		AccessToken:  t.AccessToken,
		CharacterID:  int64(t.CharacterID),
		ExpiresAt:    t.ExpiresAt,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
	}
	err := s.q.CreateToken(ctx, arg)
	if err != nil {
		if !isSqlite3ErrConstraint(err) {
			return err
		}
		arg := repository.UpdateTokenParams{
			AccessToken:  t.AccessToken,
			CharacterID:  int64(t.CharacterID),
			ExpiresAt:    t.ExpiresAt,
			RefreshToken: t.RefreshToken,
			TokenType:    t.TokenType,
		}
		if err := s.q.UpdateToken(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

// EnsureValid will automatically try to refresh a token that is already or about to become invalid.
func (s *Service) EnsureValid(t *Token) error {
	if !t.RemainsValid(time.Second * 60) {
		slog.Debug("Need to refresh token", "characterID", t.CharacterID)
		rawToken, err := sso.RefreshToken(s.httpClient, t.RefreshToken)
		if err != nil {
			return err
		}
		t.AccessToken = rawToken.AccessToken
		t.RefreshToken = rawToken.RefreshToken
		t.ExpiresAt = rawToken.ExpiresAt
		err = s.UpdateOrCreateToken(t)
		if err != nil {
			return err
		}
		slog.Info("Token refreshed", "characterID", t.CharacterID)
	}
	return nil
}

func (t *Token) NewContext() context.Context {
	ctx := context.WithValue(context.Background(), goesi.ContextAccessToken, t.AccessToken)
	return ctx
}
