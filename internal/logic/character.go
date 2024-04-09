package logic

import (
	"context"
	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/model"
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
		ID:            charID,
		Name:          charEsi.Name,
		CorporationID: charEsi.CorporationId,
	}
	if err = character.Save(); err != nil {
		return nil, err
	}
	token := model.Token{
		AccessToken:  ssoToken.AccessToken,
		Character:    character,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		TokenType:    ssoToken.TokenType,
	}
	if err = token.Save(); err != nil {
		return nil, err
	}
	return &token, nil
}
