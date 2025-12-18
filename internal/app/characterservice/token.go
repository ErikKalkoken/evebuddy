package characterservice

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

// HasTokenWithScopes reports whether a character's token has the requested scopes.
func (s *CharacterService) HasTokenWithScopes(ctx context.Context, characterID int32, scopes set.Set[string]) (bool, error) {
	missing, err := s.MissingScopes(ctx, characterID, scopes)
	if err != nil {
		return false, err
	}
	return missing.Size() == 0, nil
}

// CharactersWithMissingScopes returns a list of characters which are missing scopes (if any),
func (s *CharacterService) CharactersWithMissingScopes(ctx context.Context) ([]*app.EntityShort[int32], error) {
	var characters []*app.EntityShort[int32]
	cc, err := s.st.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range cc {
		missing, err := s.MissingScopes(ctx, c.ID, app.Scopes())
		if err != nil {
			return nil, err
		}
		if missing.Size() > 0 {
			characters = append(characters, c)
		}
	}
	return characters, nil
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
	token, err := s.st.GetCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureValidCharacterToken(ctx, token); err != nil {
		return nil, err
	}
	return token, nil
}

func (s *CharacterService) GetValidCharacterTokenWithScopes(ctx context.Context, characterID int32, scopes set.Set[string]) (*app.CharacterToken, error) {
	token, err := s.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if !token.HasScopes(scopes) {
		return nil, app.ErrNotFound
	}
	return token, nil
}

// ensureValidCharacterToken will automatically try to refresh a token that is already or about to become invalid.
func (s *CharacterService) ensureValidCharacterToken(ctx context.Context, token *app.CharacterToken) error {
	if token.RemainsValid(time.Second * 60) {
		return nil
	}
	slog.Debug("Need to refresh token", "characterID", token.CharacterID)
	token2 := &eveauth.Token{
		AccessToken:  token.AccessToken,
		CharacterID:  token.CharacterID,
		ExpiresAt:    token.ExpiresAt,
		RefreshToken: token.RefreshToken,
		Scopes:       token.Scopes.Slice(),
		TokenType:    token.TokenType,
	}
	err := s.authClient.RefreshToken(ctx, token2)
	if err != nil {
		return err
	}
	arg := storage.UpdateOrCreateCharacterTokenParamsFromToken(token)
	arg.AccessToken = token2.AccessToken
	arg.RefreshToken = token2.RefreshToken
	arg.ExpiresAt = token2.ExpiresAt
	err = s.st.UpdateOrCreateCharacterToken(ctx, arg)
	if err != nil {
		return err
	}
	token.AccessToken = token2.AccessToken
	token.RefreshToken = token2.RefreshToken
	token.ExpiresAt = token2.ExpiresAt
	slog.Info("Token refreshed", "characterID", token.CharacterID)
	return nil
}
