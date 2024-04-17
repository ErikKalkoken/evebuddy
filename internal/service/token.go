package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/antihax/goesi"

	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/model"
)

// getValidToken returns a valid token for a character. Convenience function.
func (s *Service) getValidToken(ctx context.Context, characterID int32) (model.Token, error) {
	t, err := s.r.GetToken(ctx, characterID)
	if err != nil {
		return model.Token{}, err
	}
	if err := s.ensureValidToken(ctx, &t); err != nil {
		return model.Token{}, err
	}
	return t, nil
}

// ensureValidToken will automatically try to refresh a token that is already or about to become invalid.
func (s *Service) ensureValidToken(ctx context.Context, t *model.Token) error {
	if !t.RemainsValid(time.Second * 60) {
		slog.Debug("Need to refresh token", "characterID", t.CharacterID)
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

// contextWithToken returns a new context with the ESI access token included
// so it can be used to authenticate requests with the goesi library.
func contextWithToken(ctx context.Context, accessToken string) context.Context {
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, accessToken)
	return ctx
}
