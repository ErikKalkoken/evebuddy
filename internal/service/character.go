package service

import (
	"context"
	"database/sql"
	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/repository"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"github.com/antihax/goesi"
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
	MailUpdatedAt  time.Time
	Name           string
	SecurityStatus float64
	SkillPoints    int
	WalletBalance  float64
}

func (c *Character) HasAlliance() bool {
	return c.Alliance.ID != 0
}

func (c *Character) HasFaction() bool {
	return c.Faction.ID != 0
}

// PortraitURL returns an image URL for a portrait of a character
func (c *Character) PortraitURL(size int) (fyne.URI, error) {
	return images.CharacterPortraitURL(int32(c.ID), size)
}

func (c *Character) ToDBUpdateParams() repository.UpdateCharacterParams {
	var allianceID, factionID sql.NullInt64
	if c.HasAlliance() {
		allianceID.Int64 = int64(c.Alliance.ID)
		allianceID.Valid = true
	}
	if c.HasFaction() {
		factionID.Int64 = int64(c.Alliance.ID)
		factionID.Valid = true
	}
	var mailUpdatedAt sql.NullTime
	if !c.MailUpdatedAt.IsZero() {
		mailUpdatedAt.Time = c.MailUpdatedAt
		mailUpdatedAt.Valid = true
	}
	var skillPoints sql.NullInt64
	if c.SkillPoints != 0 {
		skillPoints.Int64 = int64(c.SkillPoints)
		skillPoints.Valid = true
	}
	var walletBallance sql.NullFloat64
	if c.WalletBalance != 0 {
		walletBallance.Float64 = c.WalletBalance
		walletBallance.Valid = true
	}
	return repository.UpdateCharacterParams{
		AllianceID:     allianceID,
		CorporationID:  int64(c.Corporation.ID),
		Description:    c.Description,
		FactionID:      factionID,
		MailUpdatedAt:  mailUpdatedAt,
		Name:           c.Name,
		SecurityStatus: c.SecurityStatus,
		SkillPoints:    skillPoints,
		WalletBalance:  walletBallance,
	}
}

func characterFromDBModel(character repository.Character, corporation repository.EveEntity, alliance repository.EveEntity, faction repository.EveEntity) Character {
	var mailUpdateAt time.Time
	if character.MailUpdatedAt.Valid {
		mailUpdateAt = character.MailUpdatedAt.Time
	}
	return Character{
		Alliance:       eveEntityFromDBModel(alliance),
		Birthday:       character.Birthday,
		Corporation:    eveEntityFromDBModel(corporation),
		Description:    character.Description,
		Faction:        eveEntityFromDBModel(faction),
		Gender:         character.Gender,
		ID:             int32(character.ID),
		MailUpdatedAt:  mailUpdateAt,
		Name:           character.Name,
		SecurityStatus: character.SecurityStatus,
		SkillPoints:    int(character.SkillPoints.Int64),
		WalletBalance:  character.WalletBalance.Float64,
	}
}

func (s *Service) GetCharacter(id int) (Character, error) {
	row, err := s.queries.GetCharacter(context.Background(), int64(id))
	if err != nil {
		return Character{}, err
	}
	c := characterFromDBModel(row.Character, row.EveEntity, row.EveEntity_2, row.EveEntity_3)
	return c, nil
}

func (s *Service) DeleteCharacter(c *Character) error {
	return s.queries.DeleteCharacter(context.Background(), int64(c.ID))
}

func (s *Service) ListCharacters() ([]Character, error) {
	row, err := s.queries.ListCharacters(context.Background())
	if err != nil {
		return nil, err
	}
	cc := make([]Character, len(row))
	for i, charDB := range row {
		cc[i] = characterFromDBModel(charDB.Character, charDB.EveEntity, charDB.EveEntity_2, charDB.EveEntity_3)
	}
	return cc, nil
}

func (s *Service) GetFirstCharacter() (Character, error) {
	row, err := s.queries.GetFirstCharacter(context.Background())
	if err != nil {
		return Character{}, err
	}
	return characterFromDBModel(row.Character, row.EveEntity, row.EveEntity_2, row.EveEntity_3), nil
}

// CreateOrUpdateCharacterFromSSO creates or updates a character via SSO authentication.
func (s *Service) CreateOrUpdateCharacterFromSSO(ctx context.Context) error {
	ssoToken, err := sso.Authenticate(ctx, s.httpClient, esiScopes)
	if err != nil {
		return err
	}
	charID := ssoToken.CharacterID
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
	character := repository.UpdateOrCreateCharacterParams{
		Birthday:       charEsi.Birthday,
		CorporationID:  int64(charEsi.CorporationId),
		Description:    charEsi.Description,
		Gender:         charEsi.Gender,
		ID:             int64(charID),
		Name:           charEsi.Name,
		SecurityStatus: float64(charEsi.SecurityStatus),
	}
	if charEsi.AllianceId != 0 {
		character.AllianceID.Int64 = int64(charEsi.AllianceId)
		character.AllianceID.Valid = true
	}
	if charEsi.FactionId != 0 {
		character.FactionID.Int64 = int64(charEsi.FactionId)
		character.FactionID.Valid = true
	}
	token := Token{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  charID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		TokenType:    ssoToken.TokenType,
	}
	ctx2 := context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx2, charID, nil)
	if err != nil {
		slog.Error("Failed to fetch skills", "error", err)
	} else {
		character.SkillPoints.Int64 = skills.TotalSp
		character.SkillPoints.Valid = true
	}
	balance, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(ctx2, charID, nil)
	if err != nil {
		slog.Error("Failed to fetch wallet balance", "error", err)
	} else {
		character.WalletBalance.Float64 = balance
		character.WalletBalance.Valid = true
	}
	_, err = s.queries.UpdateOrCreateCharacter(ctx2, character)
	if err != nil {
		return err
	}
	if err = s.UpdateOrCreateToken(&token); err != nil {
		return err
	}
	return nil
}
