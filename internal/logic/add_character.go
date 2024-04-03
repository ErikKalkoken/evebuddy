package logic

import (
	"context"
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/api/sso"
	"example/esiapp/internal/model"
	"net/http"
	"time"
)

var esiScopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-search.search_structures.v1",
}

var httpClient = &http.Client{
	Timeout: time.Second * 30, // Timeout after 30 seconds
}

// AddCharacter adds a new character via SSO authentication and returns the new token.
func AddCharacter(ctx context.Context) (*model.Token, error) {
	ssoToken, err := sso.Authenticate(ctx, httpClient, esiScopes)
	if err != nil {
		return nil, err
	}
	charID := ssoToken.CharacterID
	charEsi, err := esi.FetchCharacter(httpClient, charID)
	if err != nil {
		return nil, err
	}
	ids := []int32{charID, charEsi.CorporationID}
	if charEsi.AllianceID != 0 {
		ids = append(ids, charEsi.AllianceID)
	}
	if charEsi.FactionID != 0 {
		ids = append(ids, charEsi.FactionID)
	}
	_, err = AddMissingEveEntities(ids)
	if err != nil {
		return nil, err
	}
	character := model.Character{
		ID:            charID,
		Name:          charEsi.Name,
		CorporationID: charEsi.CorporationID,
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
