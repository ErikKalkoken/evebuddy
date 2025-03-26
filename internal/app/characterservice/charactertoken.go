package characterservice

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

var esiScopes = []string{
	"esi-assets.read_assets.v1",
	"esi-characters.read_contacts.v1",
	"esi-characters.read_notifications.v1",
	"esi-contracts.read_character_contracts.v1",
	"esi-clones.read_clones.v1",
	"esi-clones.read_implants.v1",
	"esi-location.read_location.v1",
	"esi-location.read_online.v1",
	"esi-location.read_ship_type.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-planets.manage_planets.v1",
	"esi-search.search_structures.v1",
	"esi-skills.read_skills.v1",
	"esi-skills.read_skillqueue.v1",
	"esi-universe.read_structures.v1",
	"esi-wallet.read_character_wallet.v1",
}

// CharacterHasTokenWithScopes reports wether a token with the requested scopes exists for a character.
func (s *CharacterService) CharacterHasTokenWithScopes(ctx context.Context, characterID int32) (bool, error) {
	t, err := s.st.GetCharacterToken(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	incoming := set.NewFromSlice(t.Scopes)
	required := set.NewFromSlice(esiScopes)
	return required.IsSubset(incoming), nil
}

// getValidCharacterToken returns a valid token for a character. Convenience function.
func (s *CharacterService) getValidCharacterToken(ctx context.Context, characterID int32) (*app.CharacterToken, error) {
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
	if !t.RemainsValid(time.Second * 60) {
		slog.Debug("Need to refresh token", "characterID", t.CharacterID)
		rawToken, err := s.SSOService.RefreshToken(ctx, t.RefreshToken)
		if err != nil {
			return err
		}
		t.AccessToken = rawToken.AccessToken
		t.RefreshToken = rawToken.RefreshToken
		t.ExpiresAt = rawToken.ExpiresAt
		err = s.st.UpdateOrCreateCharacterToken(ctx, t)
		if err != nil {
			return err
		}
		slog.Info("Token refreshed", "characterID", t.CharacterID)
	}
	return nil
}
