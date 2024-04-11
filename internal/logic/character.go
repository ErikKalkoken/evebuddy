package logic

import (
	"context"
	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/model"
	"log/slog"
)

// AddCharacter adds a new character via SSO authentication and returns the new token.
func AddCharacter(ctx context.Context) (*model.Token, error) {
	ssoToken, err := sso.Authenticate(ctx, httpClient, esiScopes)
	if err != nil {
		return nil, err
	}
	charID := ssoToken.CharacterID
	charEsi, _, err := esiClient.ESI.CharacterApi.GetCharactersCharacterId(context.Background(), charID, nil)
	if err != nil {
		return nil, err
	}
	ids := []int32{charID, charEsi.CorporationId}
	if charEsi.AllianceId != 0 {
		ids = append(ids, charEsi.AllianceId)
	}
	if charEsi.FactionId != 0 {
		ids = append(ids, charEsi.FactionId)
	}
	_, err = AddMissingEveEntities(ids)
	if err != nil {
		return nil, err
	}
	character := model.Character{
		Birthday:       charEsi.Birthday,
		CorporationID:  charEsi.CorporationId,
		Description:    charEsi.Description,
		Gender:         charEsi.Gender,
		ID:             charID,
		Name:           charEsi.Name,
		SecurityStatus: charEsi.SecurityStatus,
	}
	if charEsi.AllianceId != 0 {
		character.AllianceID.Int32 = charEsi.AllianceId
		character.AllianceID.Valid = true
	}
	if charEsi.FactionId != 0 {
		character.FactionID.Int32 = charEsi.FactionId
		character.FactionID.Valid = true
	}
	token := model.Token{
		AccessToken:  ssoToken.AccessToken,
		Character:    character,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		TokenType:    ssoToken.TokenType,
	}
	balance, _, err := esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(newContextWithToken(&token), charID, nil)
	if err != nil {
		slog.Error("Failed to fetch wallet balance", "error", err)
	} else {
		character.WalletBalance.Float64 = balance
		character.WalletBalance.Valid = true
	}
	if err = character.Save(); err != nil {
		return nil, err
	}
	if err = token.Save(); err != nil {
		return nil, err
	}
	return &token, nil
}
