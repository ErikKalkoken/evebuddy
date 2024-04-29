package service

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2/data/binding"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/api/sso"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

var esiScopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-location.read_location.v1",
	"esi-location.read_online.v1",
	"esi-location.read_ship_type.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-search.search_structures.v1",
	"esi-skills.read_skills.v1",
	"esi-wallet.read_character_wallet.v1",
}

func (s *Service) DeleteMyCharacter(characterID int32) error {
	return s.r.DeleteMyCharacter(context.Background(), characterID)
}

func (s *Service) GetMyCharacter(characterID int32) (model.MyCharacter, error) {
	return s.r.GetMyCharacter(context.Background(), characterID)
}

func (s *Service) GetAnyMyCharacter() (model.MyCharacter, error) {
	return s.r.GetFirstMyCharacter(context.Background())
}

func (s *Service) ListMyCharacters() ([]model.MyCharacterShort, error) {
	return s.r.ListMyCharacters(context.Background())
}

// UpdateOrCreateMyCharacterFromSSO creates or updates a character via SSO authentication.
func (s *Service) UpdateOrCreateMyCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) error {
	ssoToken, err := sso.Authenticate(ctx, s.httpClient, esiScopes)
	if err != nil {
		return err
	}
	infoText.Set("Fetching character from server. Please wait...")
	charID := ssoToken.CharacterID
	token := model.Token{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  charID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		TokenType:    ssoToken.TokenType,
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	_, err = s.getOrCreateEveCharacterESI(ctx, token.CharacterID)
	if err != nil {
		return err
	}
	myCharacter := model.MyCharacter{
		ID: token.CharacterID,
	}
	if err := s.updateMyCharacterESI(ctx, &myCharacter); err != nil {
		return err
	}
	if err := s.r.UpdateOrCreateMyCharacter(ctx, &myCharacter); err != nil {
		return err
	}
	if err := s.r.UpdateOrCreateToken(ctx, &token); err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateMyCharacter(characterID int32) error {
	ctx := context.Background()
	key := fmt.Sprintf("UpdateMyCharacter-%d", characterID)
	_, err, _ := s.singleGroup.Do(key, func() (any, error) {
		err := s.updateMyCharacter(ctx, characterID)
		return struct{}{}, err
	})
	return err
}

func (s *Service) updateMyCharacter(ctx context.Context, characterID int32) error {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	c, err := s.r.GetMyCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	if err := s.updateMyCharacterESI(ctx, &c); err != nil {
		return err
	}
	if err := s.r.UpdateOrCreateMyCharacter(ctx, &c); err != nil {
		return err
	}
	slog.Info("Finished updating character", "characterID", characterID)
	return nil
}

// updateMyCharacterESI updates character details and related information from ESI.
func (s *Service) updateMyCharacterESI(ctx context.Context, c *model.MyCharacter) error {
	g := new(errgroup.Group)
	g.Go(func() error {
		skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		c.SkillPoints = int(skills.TotalSp)
		return nil
	})
	g.Go(func() error {
		balance, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		c.WalletBalance = balance
		return nil
	})
	g.Go(func() error {
		online, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdOnline(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		c.LastLoginAt = online.LastLogin
		return nil
	})
	g.Go(func() error {
		r, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdLocation(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		location, err := s.getOrCreateEveSolarSystemESI(ctx, r.SolarSystemId)
		if err != nil {
			return err
		}
		c.Location = location
		return nil
	})
	g.Go(func() error {
		ship, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdShip(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		x, err := s.getOrCreateEveTypeESI(ctx, ship.ShipTypeId)
		if err != nil {
			return err
		}
		c.Ship = x
		return nil
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to update MyCharacter %d: %w", c.ID, err)
	}
	s.SectionSetUpdated(c.ID, UpdateSectionMyCharacter)
	return nil
}
