package service

import (
	"context"
	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/repository"
)

func (s *Service) DeleteCharacter(c *repository.Character) error {
	return s.r.DeleteCharacter(context.Background(), c)
}

func (s *Service) GetCharacter(id int32) (repository.Character, error) {
	return s.r.GetCharacter(context.Background(), id)
}

func (s *Service) GetFirstCharacter() (repository.Character, error) {
	return s.r.GetFirstCharacter(context.Background())
}

func (s *Service) ListCharacters() ([]repository.Character, error) {
	return s.r.ListCharacters(context.Background())
}

// UpdateOrCreateCharacterFromSSO creates or updates a character via SSO authentication.
func (s *Service) UpdateOrCreateCharacterFromSSO(ctx context.Context) error {
	ssoToken, err := sso.Authenticate(ctx, s.httpClient, esiScopes)
	if err != nil {
		return err
	}
	charID := ssoToken.CharacterID
	token := repository.Token{
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
	_, err = s.addMissingEveEntities(ctx, ids)
	if err != nil {
		return err
	}
	corporation, err := s.r.GetEveEntity(ctx, charEsi.CorporationId)
	if err != nil {
		return err
	}
	c := repository.Character{
		Birthday:       charEsi.Birthday,
		Corporation:    corporation,
		Description:    charEsi.Description,
		Gender:         charEsi.Gender,
		ID:             charID,
		Name:           charEsi.Name,
		SecurityStatus: float64(charEsi.SecurityStatus),
	}
	if charEsi.AllianceId != 0 {
		e, err := s.r.GetEveEntity(ctx, charEsi.AllianceId)
		if err != nil {
			return err
		}
		c.Alliance = e
	}
	if charEsi.FactionId != 0 {
		e, err := s.r.GetEveEntity(ctx, charEsi.FactionId)
		if err != nil {
			return err
		}
		c.Faction = e
	}
	if err = s.r.UpdateOrCreateCharacter(ctx, &c); err != nil {
		return err
	}
	if err = s.r.UpdateOrCreateToken(ctx, &token); err != nil {
		return err
	}
	return nil
}

// TODO: Add later
// skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, charID, nil)
// if err != nil {
// 	return err
// }
// balance, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(ctx, charID, nil)
// if err != nil {
// 	return err
// }
