package service

import (
	"context"
	"log/slog"

	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/model"
)

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
	_, err = s.addMissingEveEntities(ctx, ids)
	if err != nil {
		return err
	}
	corporation, err := s.r.GetEveEntity(ctx, charEsi.CorporationId)
	if err != nil {
		return err
	}
	c := model.Character{
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
	if err = s.updateCharacter(ctx, &c); err != nil {
		return err
	}
	if err = s.r.UpdateOrCreateCharacter(ctx, &c); err != nil {
		return err
	}
	if err = s.r.UpdateOrCreateToken(ctx, &token); err != nil {
		return err
	}
	return nil
}

func (s *Service) updateCharacter(ctx context.Context, c *model.Character) error {
	skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, c.ID, nil)
	if err != nil {
		return err
	}
	balance, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(ctx, c.ID, nil)
	if err != nil {
		return err
	}
	c.SkillPoints = int(skills.TotalSp)
	c.WalletBalance = balance
	rr, _, err := s.esiClient.ESI.CharacterApi.PostCharactersAffiliation(ctx, []int32{c.ID}, nil)
	if err != nil {
		return err
	}
	if len(rr) == 0 {
		return nil
	}
	r := rr[0]
	ids := []int32{r.CorporationId}
	if r.AllianceId != 0 {
		ids = append(ids, r.AllianceId)
	}
	if r.FactionId != 0 {
		ids = append(ids, r.FactionId)
	}
	_, err = s.addMissingEveEntities(ctx, ids)
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
}

func (s *Service) UpdateCharacter(characterID int32) error {
	ctx := context.Background()
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	c, err := s.r.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	if err := s.updateCharacter(ctx, &c); err != nil {
		return err
	}
	if err := s.r.UpdateOrCreateCharacter(ctx, &c); err != nil {
		return err
	}
	slog.Info("Finished updating character", "characterID", characterID)
	return nil
}
