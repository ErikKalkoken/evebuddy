package characterservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/go-set"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

// HasTokenWithScopes reports whether a character's token has the requested scopes.
func (s *CharacterService) HasTokenWithScopes(ctx context.Context, characterID int64, scopes set.Set[string]) (bool, error) {
	missing, err := s.MissingScopes(ctx, characterID, scopes)
	if err != nil {
		return false, err
	}
	return missing.Size() == 0, nil
}

// CharactersWithMissingScopes returns a list of characters which are missing scopes (if any),
func (s *CharacterService) CharactersWithMissingScopes(ctx context.Context) ([]*app.EntityShort[int64], error) {
	var characters []*app.EntityShort[int64]
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

func (s *CharacterService) MissingScopes(ctx context.Context, characterID int64, scopes set.Set[string]) (set.Set[string], error) {
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

// ValidTokenForCorporation returns a token with a specific scope and from a member character with a specific role and matching scope.
// Will be valid when any of the given roles and scopes match.
// It can optionally ensure the token is valid by with checkToken.
// It returns [app.ErrNotFound] if no such token exists.
func (s *CharacterService) ValidTokenForCorporation(ctx context.Context, corporationID int64, roles set.Set[app.Role], scopes set.Set[string], checkToken bool) (*app.CharacterToken, error) {
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
		_, err := s.ensureValidToken(ctx, t)
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

// ValidToken returns a valid token for a character.
// Will automatically try to refresh a token if needed.
func (s *CharacterService) ValidToken(ctx context.Context, characterID int64) (*app.CharacterToken, error) {
	token, err := s.st.GetCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ensureValidToken(ctx, token); err != nil {
		return nil, err
	}
	return token, nil
}

func (s *CharacterService) GetValidCharacterTokenWithScopes(ctx context.Context, characterID int64, scopes set.Set[string]) (*app.CharacterToken, error) {
	token, err := s.ValidToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if !token.HasScopes(scopes) {
		return nil, app.ErrNotFound
	}
	return token, nil
}

// ensureValidToken will try to refresh token if it is about to become invalid
// and report whether it was refreshed.
func (s *CharacterService) ensureValidToken(ctx context.Context, token *app.CharacterToken) (bool, error) {
	const tokenTimeout = time.Second * 60
	if token.RemainsValid(tokenTimeout) {
		return false, nil
	}
	slog.Debug("Need to refresh token", "characterID", token.CharacterID)
	x, err, _ := s.sfg.Do(fmt.Sprintf("ensureValidToken-%d", token.ID), func() (any, error) {
		token2, err := s.st.GetCharacterToken(ctx, token.CharacterID)
		if err != nil {
			return nil, err
		}
		if token2.RemainsValid(tokenTimeout) {
			return token2, nil
		}
		at := token2.AuthToken()
		if err := s.authClient.RefreshToken(ctx, at); err != nil {
			return nil, err
		}
		if err = s.st.UpdateOrCreateCharacterToken(ctx, storage.UpdateOrCreateCharacterTokenParams{
			AccessToken:  at.AccessToken,
			CharacterID:  int64(at.CharacterID),
			ExpiresAt:    at.ExpiresAt,
			RefreshToken: at.RefreshToken,
			Scopes:       set.Of(at.Scopes...),
			TokenType:    at.TokenType,
		}); err != nil {
			return nil, err
		}
		slog.Info("Token refreshed", "characterID", token.CharacterID)
		token2.AccessToken = at.AccessToken
		token2.RefreshToken = at.RefreshToken
		token2.ExpiresAt = at.ExpiresAt
		return token2, err
	})
	if err != nil {
		return false, err
	}
	token2 := x.(*app.CharacterToken)
	*token = *token2
	return true, err
}

type tokenSource struct {
	ensureValid func(context.Context, *app.CharacterToken) (bool, error)

	sfg   *singleflight.Group
	token *app.CharacterToken
}

func newTokenSource(token *app.CharacterToken, ensureValid func(context.Context, *app.CharacterToken) (bool, error)) *tokenSource {
	ts := &tokenSource{
		token:       token,
		ensureValid: ensureValid,
		sfg:         new(singleflight.Group),
	}
	return ts
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	x, err, _ := ts.sfg.Do("KEY", func() (any, error) {
		if time.Now().After(ts.token.ExpiresAt) {
			_, err := ts.ensureValid(context.Background(), ts.token)
			if err != nil {
				return nil, err
			}
		}
		tok := &oauth2.Token{
			AccessToken:  ts.token.AccessToken,
			RefreshToken: ts.token.RefreshToken,
			Expiry:       ts.token.ExpiresAt,
			ExpiresIn:    int64(time.Until(ts.token.ExpiresAt).Seconds()),
		}
		return tok, nil
	})
	return x.(*oauth2.Token), err
}
