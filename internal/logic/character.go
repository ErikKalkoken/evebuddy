package logic

import (
	"context"
	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/model"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
)

// An Eve Online character.
type Character struct {
	Alliance       EveEntity
	Birthday       time.Time
	Corporation    EveEntity
	Description    string
	Faction        EveEntity
	Gender         string
	ID             int32
	Name           string
	SecurityStatus float32
	SkillPoints    int
	WalletBalance  float64
}

func (c *Character) Delete() error {
	return model.DeleteCharacter(c.ID)
}

func (c *Character) HasAlliance() bool {
	return c.Alliance.ID != 0
}

func (c *Character) HasFaction() bool {
	return c.Faction.ID != 0
}

// PortraitURL returns an image URL for a portrait of a character
func (c *Character) PortraitURL(size int) (fyne.URI, error) {
	return images.CharacterPortraitURL(c.ID, size)
}

func characterFromDBModel(c model.Character) Character {
	return Character{
		Alliance:       eveEntityFromDBModel(c.Alliance),
		Birthday:       c.Birthday,
		Corporation:    eveEntityFromDBModel(c.Corporation),
		Description:    c.Description,
		Faction:        eveEntityFromDBModel(c.Faction),
		Gender:         c.Gender,
		ID:             c.ID,
		Name:           c.Name,
		SecurityStatus: c.SecurityStatus,
		SkillPoints:    int(c.SkillPoints.Int64),
		WalletBalance:  c.WalletBalance.Float64,
	}
}

func GetCharacter(id int32) (Character, error) {
	charDB, err := model.GetCharacter(id)
	if err != nil {
		return Character{}, err
	}
	err = charDB.GetAlliance()
	if err != nil {
		slog.Error(err.Error())
		return Character{}, err
	}
	err = charDB.GetFaction()
	if err != nil {
		slog.Error(err.Error())
		return Character{}, err
	}
	c := characterFromDBModel(charDB)
	return c, nil
}

func ListCharacters() ([]Character, error) {
	charsDB, err := model.ListCharacters()
	if err != nil {
		return nil, err
	}
	cc := make([]Character, len(charsDB))
	for i, charDB := range charsDB {
		cc[i] = characterFromDBModel(charDB)
	}
	return cc, nil
}

func GetFirstCharacter() (Character, error) {
	charDB, err := model.GetFirstCharacter()
	if err != nil {
		return Character{}, err
	}
	return characterFromDBModel(charDB), nil
}

// AddCharacter adds a new character via SSO authentication and returns the new token.
func AddCharacter(ctx context.Context) error {
	ssoToken, err := sso.Authenticate(ctx, httpClient, esiScopes)
	if err != nil {
		return err
	}
	charID := ssoToken.CharacterID
	charEsi, _, err := esiClient.ESI.CharacterApi.GetCharactersCharacterId(context.Background(), charID, nil)
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
	_, err = addMissingEveEntities(ids)
	if err != nil {
		return err
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
	skills, _, err := esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(newContextWithToken(&token), charID, nil)
	if err != nil {
		slog.Error("Failed to fetch skills", "error", err)
	} else {
		character.SkillPoints.Int64 = skills.TotalSp
		character.SkillPoints.Valid = true
	}
	balance, _, err := esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(newContextWithToken(&token), charID, nil)
	if err != nil {
		slog.Error("Failed to fetch wallet balance", "error", err)
	} else {
		character.WalletBalance.Float64 = balance
		character.WalletBalance.Valid = true
	}
	if err = character.Save(); err != nil {
		return err
	}
	if err = token.Save(); err != nil {
		return err
	}
	return nil
}
