package characterservice

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// HasTokenWithScopes reports whether a character's token has the requested scopes.
func (s *CharacterService) HasTokenWithScopes(ctx context.Context, characterID int32) (bool, error) {
	t, err := s.st.GetCharacterToken(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	current := set.Of(t.Scopes...)
	required := app.Scopes()
	hasScope := current.ContainsAll(required.All())
	return hasScope, nil
}

func (s *CharacterService) ValidCharacterTokenForCorporation(ctx context.Context, corporationID int32, role app.Role) (*app.CharacterToken, error) {
	token, err := s.st.ListCharacterTokenForCorporation(ctx, corporationID, role)
	if err != nil {
		return nil, err
	}
	for _, t := range token {
		err := s.ensureValidCharacterToken(ctx, t)
		if err != nil {
			slog.Error("Failed to refresh token for corporation", "characterID", t.CharacterID, "corporationID", corporationID, "role", role)
			continue
		}
		return t, nil
	}
	return nil, app.ErrNotFound
}

// GetValidCharacterToken returns a valid token for a character.
// Will automatically try to refresh a token if needed.
func (s *CharacterService) GetValidCharacterToken(ctx context.Context, characterID int32) (*app.CharacterToken, error) {
	t, err := s.st.GetCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureValidCharacterToken(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ensureValidCharacterToken will automatically try to refresh a token that is already or about to become invalid.
func (s *CharacterService) ensureValidCharacterToken(ctx context.Context, t *app.CharacterToken) error {
	if t.RemainsValid(time.Second * 60) {
		return nil
	}
	slog.Debug("Need to refresh token", "characterID", t.CharacterID)
	rawToken, err := s.sso.RefreshToken(ctx, t.RefreshToken)
	if err != nil {
		return err
	}
	err = s.st.UpdateOrCreateCharacterToken(ctx, storage.UpdateOrCreateCharacterTokenParams{
		AccessToken:  rawToken.AccessToken,
		CharacterID:  t.CharacterID,
		ExpiresAt:    rawToken.ExpiresAt,
		RefreshToken: rawToken.RefreshToken,
		Scopes:       t.Scopes,
		TokenType:    t.TokenType,
	})
	if err != nil {
		return err
	}
	slog.Info("Token refreshed", "characterID", t.CharacterID)
	return nil
}
