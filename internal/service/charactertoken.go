package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/antihax/goesi"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/sso"
	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

var esiScopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-clones.read_clones.v1",
	"esi-clones.read_implants.v1",
	"esi-location.read_location.v1",
	"esi-location.read_online.v1",
	"esi-location.read_ship_type.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-search.search_structures.v1",
	"esi-skills.read_skills.v1",
	"esi-skills.read_skillqueue.v1",
	"esi-universe.read_structures.v1",
	"esi-wallet.read_character_wallet.v1",
}

// CharacterHasTokenWithScopes reports wether a token with the requested scopes exists for a character.
func (s *Service) CharacterHasTokenWithScopes(characterID int32) (bool, error) {
	ctx := context.Background()
	t, err := s.r.GetCharacterToken(ctx, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	got := set.NewFromSlice(t.Scopes)
	want := set.NewFromSlice(esiScopes)
	return got.Equal(want), nil
}

// getValidCharacterToken returns a valid token for a character. Convenience function.
func (s *Service) getValidCharacterToken(ctx context.Context, characterID int32) (*model.CharacterToken, error) {
	t, err := s.r.GetCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureValidCharacterToken(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ensureValidCharacterToken will automatically try to refresh a token that is already or about to become invalid.
func (s *Service) ensureValidCharacterToken(ctx context.Context, t *model.CharacterToken) error {
	if !t.RemainsValid(time.Second * 60) {
		slog.Debug("Need to refresh token", "characterID", t.CharacterID)
		rawToken, err := sso.RefreshToken(s.httpClient, t.RefreshToken)
		if err != nil {
			return err
		}
		t.AccessToken = rawToken.AccessToken
		t.RefreshToken = rawToken.RefreshToken
		t.ExpiresAt = rawToken.ExpiresAt
		err = s.r.UpdateOrCreateCharacterToken(ctx, t)
		if err != nil {
			return err
		}
		slog.Info("Token refreshed", "characterID", t.CharacterID)
	}
	return nil
}

// contextWithESIToken returns a new context with the ESI access token included
// so it can be used to authenticate requests with the goesi library.
func contextWithESIToken(ctx context.Context, accessToken string) context.Context {
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, accessToken)
	return ctx
}