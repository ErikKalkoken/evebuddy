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
func (s *CharacterService) HasTokenWithScopes(ctx context.Context, characterID int32, scopes set.Set[string]) (bool, error) {
	missing, err := s.MissingScopes(ctx, characterID, scopes)
	if err != nil {
		return false, err
	}
	return missing.Size() == 0, nil
}

func (s *CharacterService) MissingScopes(ctx context.Context, characterID int32, scopes set.Set[string]) (set.Set[string], error) {
	var missing set.Set[string]
	t, err := s.st.GetCharacterToken(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		return scopes, nil
	}
	if err != nil {
		return missing, err
	}
	return set.Difference(scopes, t.Scopes), nil
}

// CharacterTokenForCorporation returns a token with a specific scope and from a member character with a specific role and matching scope.
// Will be valid when any of the given roles and scopes match.
// It can optionally ensure the token is valid by with checkToken.
// It returns [app.ErrNotFound] if no such token exists.
func (s *CharacterService) CharacterTokenForCorporation(ctx context.Context, corporationID int32, roles set.Set[app.Role], scopes set.Set[string], checkToken bool) (*app.CharacterToken, error) {
	token, err := s.st.ListCharacterTokenForCorporation(ctx, corporationID, roles, scopes)
	if err != nil {
		return nil, err
	}
	if len(token) == 0 {
		return nil, app.ErrNotFound
	}
	if !checkToken {
		return token[0], nil
	}
	for _, t := range token {
		err := s.ensureValidCharacterToken(ctx, t)
		if err != nil {
			slog.Error(
				"Failed to refresh token for corporation",
				"characterID", t.CharacterID,
				"corporationID", corporationID,
				"roles", roles,
				"scopes", scopes,
			)
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
	arg := storage.UpdateOrCreateCharacterTokenParamsFromToken(t)
	arg.AccessToken = rawToken.AccessToken
	arg.RefreshToken = rawToken.RefreshToken
	arg.ExpiresAt = rawToken.ExpiresAt
	err = s.st.UpdateOrCreateCharacterToken(ctx, arg)
	if err != nil {
		return err
	}
	t.AccessToken = rawToken.AccessToken
	t.RefreshToken = rawToken.RefreshToken
	t.ExpiresAt = rawToken.ExpiresAt
	slog.Info("Token refreshed", "characterID", t.CharacterID)
	return nil
}
