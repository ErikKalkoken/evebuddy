package service

import (
	"context"
	"database/sql"
	"errors"
	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/sqlc"

	"github.com/antihax/goesi"
)

// UpdateOrCreateCharacterFromSSO creates or updates a character via SSO authentication.
func (s *Service) UpdateOrCreateCharacterFromSSO(ctx context.Context) error {
	ssoToken, err := sso.Authenticate(ctx, s.httpClient, esiScopes)
	if err != nil {
		return err
	}
	charID := ssoToken.CharacterID
	token := Token{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  charID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		TokenType:    ssoToken.TokenType,
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
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
	_, err = s.addMissingEveEntities(ids)
	if err != nil {
		return err
	}
	_, err = s.r.GetCharacter(ctx, charID)
	if errors.Is(err, sql.ErrNoRows) {
		arg := sqlc.CreateCharacterParams{
			Birthday:       charEsi.Birthday,
			CorporationID:  int64(charEsi.CorporationId),
			Description:    charEsi.Description,
			Gender:         charEsi.Gender,
			ID:             int64(charID),
			Name:           charEsi.Name,
			SecurityStatus: float64(charEsi.SecurityStatus),
		}
		if charEsi.AllianceId != 0 {
			arg.AllianceID.Int64 = int64(charEsi.AllianceId)
			arg.AllianceID.Valid = true
		}
		if charEsi.FactionId != 0 {
			arg.FactionID.Int64 = int64(charEsi.FactionId)
			arg.FactionID.Valid = true
		}
		_, err := s.r.CreateCharacter(ctx, arg)
		if err != nil {
			return err
		}
	} else {
		arg := sqlc.UpdateCharacterParams{
			CorporationID:  int64(charEsi.CorporationId),
			Description:    charEsi.Description,
			Name:           charEsi.Name,
			SecurityStatus: float64(charEsi.SecurityStatus),
		}
		if charEsi.AllianceId != 0 {
			arg.AllianceID.Int64 = int64(charEsi.AllianceId)
			arg.AllianceID.Valid = true
		}
		if charEsi.FactionId != 0 {
			arg.FactionID.Int64 = int64(charEsi.FactionId)
			arg.FactionID.Valid = true
		}
		err := s.r.UpdateCharacter(ctx, arg)
		if err != nil {
			return err
		}
	}
	if err = s.UpdateOrCreateToken(&token); err != nil {
		return err
	}
	return nil
}

// TODO: Implement again
// func (s *Service) UpdateCharacter(c *Character) {
// 	skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, charID, nil)
// 	var skillPoints sql.NullInt64
// 	if err != nil {
// 		slog.Error("Failed to fetch skills", "error", err)
// 	} else {
// 		skillPoints.Int64 = skills.TotalSp
// 		skillPoints.Valid = true
// 	}
// 	balance, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(ctx, charID, nil)
// 	var walletBalance sql.NullFloat64
// 	if err != nil {
// 		slog.Error("Failed to fetch wallet balance", "error", err)
// 	} else {
// 		walletBalance.Float64 = balance
// 		walletBalance.Valid = true
// 	}
// }
