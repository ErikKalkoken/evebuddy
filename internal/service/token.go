package service

import (
	"context"
	"log/slog"
	"time"

	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/repository"
)

// GetValidToken returns a valid token for a character. Convenience function.
func (s *Service) GetValidToken(ctx context.Context, characterID int32) (repository.Token, error) {
	t, err := s.r.GetToken(ctx, characterID)
	if err != nil {
		return repository.Token{}, err
	}
	if err := s.EnsureValidToken(ctx, &t); err != nil {
		return repository.Token{}, err
	}
	return t, nil
}

// EnsureValidToken will automatically try to refresh a token that is already or about to become invalid.
func (s *Service) EnsureValidToken(ctx context.Context, t *repository.Token) error {
	if !t.RemainsValid(time.Second * 60) {
		slog.Debug("Need to refresh token", "characterID", t.CharacterID)
		ctx := context.Background()
		rawToken, err := sso.RefreshToken(s.httpClient, t.RefreshToken)
		if err != nil {
			return err
		}
		t.AccessToken = rawToken.AccessToken
		t.RefreshToken = rawToken.RefreshToken
		t.ExpiresAt = rawToken.ExpiresAt
		err = s.r.UpdateOrCreateToken(ctx, t)
		if err != nil {
			return err
		}
		slog.Info("Token refreshed", "characterID", t.CharacterID)
	}
	return nil
}
