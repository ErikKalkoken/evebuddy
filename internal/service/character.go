package service

import (
	"context"
	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/repository"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2/data/binding"
)

func (s *Service) DeleteCharacter(characterID int32) error {
	return s.r.DeleteCharacter(context.Background(), characterID)
}

func (s *Service) GetCharacter(characterID int32) (repository.Character, error) {
	return s.r.GetCharacter(context.Background(), characterID)
}

func (s *Service) GetAnyCharacter() (repository.Character, error) {
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

func (s *Service) updateCharacter(ctx context.Context, c *repository.Character) error {
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
	var alliance repository.EveEntity
	if r.AllianceId != 0 {
		alliance, err = s.r.GetEveEntity(ctx, r.AllianceId)
		if err != nil {
			return err
		}
	}
	c.Alliance = alliance
	var faction repository.EveEntity
	if r.FactionId != 0 {
		faction, err = s.r.GetEveEntity(ctx, r.FactionId)
		if err != nil {
			return err
		}
	}
	c.Faction = faction
	return nil
}

func (s *Service) StartCharacterUpdateTask(statusBarTex binding.String) {
	ticker := time.NewTicker(120 * time.Second)
	go func() {
		for {
			s.updateCharacters(statusBarTex)
			<-ticker.C
		}
	}()
}

func (s *Service) updateCharacters(statusBarTex binding.String) error {
	ctx := context.Background()
	cc, err := s.r.ListCharacters(ctx)
	if err != nil {
		return err
	}
	slog.Info("Start updating characters", "count", len(cc))
	for _, c := range cc {
		token, err := s.getValidToken(ctx, c.ID)
		if err != nil {
			slog.Error("Failed to update character", "characterID", c.ID, "err", err)
			continue
		}
		ctx = contextWithToken(ctx, token.AccessToken)
		if err := s.updateCharacter(ctx, &c); err != nil {
			slog.Error("Failed to update character", "characterID", c.ID, "err", err)
			continue
		}
		if err := s.r.UpdateOrCreateCharacter(ctx, &c); err != nil {
			slog.Error("Failed to update character", "characterID", c.ID, "err", err)
			continue
		}
		slog.Info("Completed updating character", "characterID", c.ID)
		statusBarTex.Set(fmt.Sprintf("Updated %d characters", len(cc)))
	}
	return nil
}
