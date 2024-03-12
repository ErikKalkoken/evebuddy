package core

import (
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
)

// AddCharacter adds a new character via SSO authentication and returns the new token.
func AddCharacter() (*storage.Token, error) {
	scopes := []string{
		"esi-characters.read_contacts.v1",
		"esi-universe.read_structures.v1",
		"esi-mail.read_mail.v1",
	}
	ssoToken, err := sso.Authenticate(httpClient, scopes)
	if err != nil {
		return nil, err
	}
	character := storage.Character{
		ID:   ssoToken.CharacterID,
		Name: ssoToken.CharacterName,
	}
	if err = character.Save(); err != nil {
		return nil, err
	}
	token := storage.Token{
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
