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
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
func (s *CharacterService) CharactersWithMissingScopes(ctx context.Context) ([]*app.EntityShort, error) {
	var characters []*app.EntityShort
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

// TokenSourceForCorporation returns a token and character ID of the token's owner
// from a member character of the corporation.
// Will match when any of the given roles and scopes.
// It returns [app.ErrNotFound] if no such token exists.
func (s *CharacterService) TokenSourceForCorporation(ctx context.Context, corporationID int64, roles set.Set[app.Role], scopes set.Set[string]) (oauth2.TokenSource, int64, error) {
	tokens, err := s.st.ListCharacterTokenForCorporation(ctx, corporationID, roles, scopes)
	if err != nil {
		return nil, 0, err
	}
	token, ok := xslices.Pop(&tokens)
	if !ok {
		return nil, 0, app.ErrNotFound
	}
	ts := newTokenSource(token, s.ensureValidToken)
	return ts, token.CharacterID, nil
}

// TokenSource returns a valid token source for a character.
// The token source will automatically refresh a token when needed.
func (s *CharacterService) TokenSource(ctx context.Context, characterID int64, scopes set.Set[string]) (oauth2.TokenSource, error) {
	token, err := s.st.GetCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if !token.HasScopes(scopes) {
		return nil, app.ErrNotFound
	}
	ts := newTokenSource(token, s.ensureValidToken)
	return ts, nil
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

	sfg   singleflight.Group
	token *app.CharacterToken
}

func newTokenSource(token *app.CharacterToken, ensureValid func(context.Context, *app.CharacterToken) (bool, error)) *tokenSource {
	ts := &tokenSource{
		token:       token,
		ensureValid: ensureValid,
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
		return ts.token.OauthToken(), nil
	})
	return x.(*oauth2.Token), err
}
