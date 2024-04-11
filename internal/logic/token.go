package logic

import (
	"context"
	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/model"
	"log/slog"
	"time"

	"github.com/antihax/goesi"
)

// FetchValidToken returns a valid token for a character. Convenience function.
func FetchValidToken(characterID int32) (*model.Token, error) {
	t, err := model.FetchToken(characterID)
	if err != nil {
		return nil, err
	}
	if err := EnsureValidToken(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

// EnsureValidToken will automatically try to refresh a token that is already or about to become invalid.
func EnsureValidToken(token *model.Token) error {
	if !token.RemainsValid(time.Second * 60) {
		slog.Debug("Need to refresh token", "characterID", token.CharacterID)
		rawToken, err := sso.RefreshToken(httpClient, token.RefreshToken)
		if err != nil {
			return err
		}
		token.AccessToken = rawToken.AccessToken
		token.RefreshToken = rawToken.RefreshToken
		token.ExpiresAt = rawToken.ExpiresAt
		err = token.Save()
		if err != nil {
			return err
		}
		slog.Info("Token refreshed", "characterID", token.CharacterID)
	}
	return nil
}

func newContextWithToken(token *model.Token) context.Context {
	ctx := context.WithValue(context.Background(), goesi.ContextAccessToken, token.AccessToken)
	return ctx
}
