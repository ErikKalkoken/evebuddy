package service

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/model"
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

func (s *Service) DeleteCharacter(characterID int32) error {
	return s.r.DeleteCharacter(context.Background(), characterID)
}

func (s *Service) GetCharacter(characterID int32) (model.Character, error) {
	return s.r.GetCharacter(context.Background(), characterID)
}

func (s *Service) GetAnyCharacter() (model.Character, error) {
	return s.r.GetFirstCharacter(context.Background())
}

func (s *Service) ListCharacters() ([]model.Character, error) {
	return s.r.ListCharacters(context.Background())
}

// UpdateOrCreateCharacterFromSSO creates or updates a character via SSO authentication.
func (s *Service) UpdateOrCreateCharacterFromSSO(ctx context.Context) error {
	ssoToken, err := sso.Authenticate(ctx, s.httpClient, esiScopes)
	if err != nil {
		return err
	}
	charID := ssoToken.CharacterID
	token := model.Token{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  charID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		TokenType:    ssoToken.TokenType,
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	charEsi, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, charID, nil)
	if err != nil {
		return err
	}
	ids := []int32{charID, charEsi.CorporationId}
	if charEsi.AllianceId != 0 {
		ids = append(ids, charEsi.AllianceId)
	}
	if charEsi.FactionId != 0 {
		ids = append(ids, charEsi.FactionId)
	}
	_, err = s.AddMissingEveEntities(ctx, ids)
	if err != nil {
		return err
	}
	corporation, err := s.r.GetEveEntity(ctx, charEsi.CorporationId)
	if err != nil {
		return err
	}
	if err := s.updateRacesESI(ctx); err != nil {
		return err
	}
	race, err := s.r.GetRace(ctx, charEsi.RaceId)
	if err != nil {
		return err
	}
	character := model.Character{
		Birthday:       charEsi.Birthday,
		Corporation:    corporation,
		Description:    charEsi.Description,
		Gender:         charEsi.Gender,
		ID:             charID,
		Name:           charEsi.Name,
		Race:           race,
		SecurityStatus: float64(charEsi.SecurityStatus),
	}
	if charEsi.AllianceId != 0 {
		e, err := s.r.GetEveEntity(ctx, charEsi.AllianceId)
		if err != nil {
			return err
		}
		character.Alliance = e
	}
	if charEsi.FactionId != 0 {
		e, err := s.r.GetEveEntity(ctx, charEsi.FactionId)
		if err != nil {
			return err
		}
		character.Faction = e
	}
	if err := s.updateCharacterDetailsESI(ctx, &character); err != nil {
		return err
	}
	if err := s.r.UpdateOrCreateCharacter(ctx, &character); err != nil {
		return err
	}
	if err := s.r.UpdateOrCreateToken(ctx, &token); err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateCharacterDetails(characterID int32) error {
	ctx := context.Background()
	key := fmt.Sprintf("UpdateCharacterDetails-%d", characterID)
	_, err, _ := s.singleGroup.Do(key, func() (interface{}, error) {
		err := s.updateCharacterDetails(ctx, characterID)
		return struct{}{}, err
	})
	return err
}

func (s *Service) updateCharacterDetails(ctx context.Context, characterID int32) error {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	c, err := s.r.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	if err := s.updateCharacterDetailsESI(ctx, &c); err != nil {
		return err
	}
	if err := s.r.UpdateOrCreateCharacter(ctx, &c); err != nil {
		return err
	}
	slog.Info("Finished updating character", "characterID", characterID)
	return nil
}

// updateCharacterDetailsESI updates character details and related information from ESI.
func (s *Service) updateCharacterDetailsESI(ctx context.Context, c *model.Character) error {
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
		locationESI, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdLocation(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		entityIDs := []int32{locationESI.SolarSystemId}
		if locationESI.StationId != 0 {
			entityIDs = append(entityIDs, locationESI.StationId)
		}
		_, err = s.AddMissingEveEntities(ctx, entityIDs)
		if err != nil {
			return err
		}
		var location model.EveEntity
		if locationESI.StationId != 0 {
			location, err = s.r.GetEveEntity(ctx, locationESI.StationId)
		} else {
			location, err = s.r.GetEveEntity(ctx, locationESI.SolarSystemId)
		}
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
		_, err = s.AddMissingEveEntities(ctx, []int32{ship.ShipTypeId})
		if err != nil {
			return err
		}
		o, err := s.r.GetEveEntity(ctx, ship.ShipTypeId)
		if err != nil {
			return err
		}
		c.Ship = o
		return nil
	})
	g.Go(func() error {
		rr, _, err := s.esiClient.ESI.CharacterApi.PostCharactersAffiliation(ctx, []int32{c.ID}, nil)
		if err != nil {
			return err
		}
		if len(rr) == 0 {
			return nil
		}
		r := rr[0]
		entityIDs := []int32{c.ID}
		entityIDs = append(entityIDs, r.CorporationId)
		if r.AllianceId != 0 {
			entityIDs = append(entityIDs, r.AllianceId)
		}
		if r.FactionId != 0 {
			entityIDs = append(entityIDs, r.FactionId)
		}
		_, err = s.AddMissingEveEntities(ctx, entityIDs)
		if err != nil {
			return err
		}
		corporation, err := s.r.GetEveEntity(ctx, r.CorporationId)
		if err != nil {
			return err
		}
		c.Corporation = corporation
		var alliance model.EveEntity
		if r.AllianceId != 0 {
			alliance, err = s.r.GetEveEntity(ctx, r.AllianceId)
			if err != nil {
				return err
			}
		}
		c.Alliance = alliance
		var faction model.EveEntity
		if r.FactionId != 0 {
			faction, err = s.r.GetEveEntity(ctx, r.FactionId)
			if err != nil {
				return err
			}
		}
		c.Faction = faction
		return nil
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to update character %d: %w", c.ID, err)
	}
	return nil
}
