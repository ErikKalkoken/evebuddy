package repository

import (
	"context"
	"database/sql"
	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/sqlc"
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

func (c *Character) ToDBUpdateParams() sqlc.UpdateCharacterParams {
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
	return sqlc.UpdateCharacterParams{
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

func characterFromDBModel(character sqlc.Character, corporation sqlc.EveEntity, alliance sqlc.EveEntity, faction sqlc.EveEntity) Character {
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

func (r *Repository) GetCharacter(ctx context.Context, id int32) (Character, error) {
	row, err := r.q.GetCharacter(ctx, int64(id))
	if err != nil {
		return Character{}, err
	}
	c := characterFromDBModel(row.Character, row.EveEntity, row.EveEntity_2, row.EveEntity_3)
	return c, nil
}

func (r *Repository) DeleteCharacter(ctx context.Context, c *Character) error {
	return r.q.DeleteCharacter(ctx, int64(c.ID))
}

func (r *Repository) ListCharacters(ctx context.Context) ([]Character, error) {
	row, err := r.q.ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	cc := make([]Character, len(row))
	for i, charDB := range row {
		cc[i] = characterFromDBModel(charDB.Character, charDB.EveEntity, charDB.EveEntity_2, charDB.EveEntity_3)
	}
	return cc, nil
}

func (r *Repository) GetFirstCharacter(ctx context.Context) (Character, error) {
	row, err := r.q.GetFirstCharacter(ctx)
	if err != nil {
		return Character{}, err
	}
	return characterFromDBModel(row.Character, row.EveEntity, row.EveEntity_2, row.EveEntity_3), nil
}
