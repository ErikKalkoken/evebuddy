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

func (r *Repository) DeleteCharacter(ctx context.Context, c *Character) error {
	return r.q.DeleteCharacter(ctx, int64(c.ID))
}

func (r *Repository) GetCharacter(ctx context.Context, id int32) (Character, error) {
	row, err := r.q.GetCharacter(ctx, int64(id))
	if err != nil {
		return Character{}, err
	}
	var mailUpdateAt time.Time
	if row.MailUpdatedAt.Valid {
		mailUpdateAt = row.MailUpdatedAt.Time
	}
	c := Character{
		Birthday: row.Birthday,
		Corporation: EveEntity{ID: int32(row.CorporationID),
			Name:     row.Name_2,
			Category: eveEntityCategoryFromDBModel(row.Category),
		},
		Description:    row.Description,
		Gender:         row.Gender,
		ID:             int32(row.ID),
		MailUpdatedAt:  mailUpdateAt,
		Name:           row.Name,
		SecurityStatus: row.SecurityStatus,
		SkillPoints:    int(row.SkillPoints.Int64),
		WalletBalance:  row.WalletBalance.Float64,
	}
	if row.AllianceID.Valid {
		c.Alliance = EveEntity{
			ID:       int32(row.AllianceID.Int64),
			Name:     row.Name_3.String,
			Category: eveEntityCategoryFromDBModel(row.Category_2.String),
		}
	}
	if row.FactionID.Valid {
		c.Faction = EveEntity{
			ID:       int32(row.FactionID.Int64),
			Name:     row.Name_4.String,
			Category: eveEntityCategoryFromDBModel(row.Category_3.String),
		}
	}
	return c, nil
}

func (r *Repository) GetFirstCharacter(ctx context.Context) (Character, error) {
	row, err := r.q.GetFirstCharacter(ctx)
	if err != nil {
		return Character{}, err
	}
	var mailUpdateAt time.Time
	if row.MailUpdatedAt.Valid {
		mailUpdateAt = row.MailUpdatedAt.Time
	}
	c := Character{
		Birthday: row.Birthday,
		Corporation: EveEntity{ID: int32(row.CorporationID),
			Name:     row.Name_2,
			Category: eveEntityCategoryFromDBModel(row.Category),
		},
		Description:    row.Description,
		Gender:         row.Gender,
		ID:             int32(row.ID),
		MailUpdatedAt:  mailUpdateAt,
		Name:           row.Name,
		SecurityStatus: row.SecurityStatus,
		SkillPoints:    int(row.SkillPoints.Int64),
		WalletBalance:  row.WalletBalance.Float64,
	}
	if row.AllianceID.Valid {
		c.Alliance = EveEntity{
			ID:       int32(row.AllianceID.Int64),
			Name:     row.Name_3.String,
			Category: eveEntityCategoryFromDBModel(row.Category_2.String),
		}
	}
	if row.FactionID.Valid {
		c.Faction = EveEntity{
			ID:       int32(row.FactionID.Int64),
			Name:     row.Name_4.String,
			Category: eveEntityCategoryFromDBModel(row.Category_3.String),
		}
	}
	return c, nil
}

func (r *Repository) ListCharacters(ctx context.Context) ([]Character, error) {
	rows, err := r.q.ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	cc := make([]Character, len(rows))
	for i, row := range rows {
		var mailUpdateAt time.Time
		if row.MailUpdatedAt.Valid {
			mailUpdateAt = row.MailUpdatedAt.Time
		}
		c := Character{
			Birthday: row.Birthday,
			Corporation: EveEntity{ID: int32(row.CorporationID),
				Name:     row.Name_2,
				Category: eveEntityCategoryFromDBModel(row.Category),
			},
			Description:    row.Description,
			Gender:         row.Gender,
			ID:             int32(row.ID),
			MailUpdatedAt:  mailUpdateAt,
			Name:           row.Name,
			SecurityStatus: row.SecurityStatus,
			SkillPoints:    int(row.SkillPoints.Int64),
			WalletBalance:  row.WalletBalance.Float64,
		}
		if row.AllianceID.Valid {
			c.Alliance = EveEntity{
				ID:       int32(row.AllianceID.Int64),
				Name:     row.Name_3.String,
				Category: eveEntityCategoryFromDBModel(row.Category_2.String),
			}
		}
		if row.FactionID.Valid {
			c.Faction = EveEntity{
				ID:       int32(row.FactionID.Int64),
				Name:     row.Name_4.String,
				Category: eveEntityCategoryFromDBModel(row.Category_3.String),
			}
		}
		cc[i] = c
	}
	return cc, nil
}

func (r *Repository) UpdateOrCreateCharacter(ctx context.Context, c *Character) error {
	arg := sqlc.CreateCharacterParams{
		Birthday:       c.Birthday,
		CorporationID:  int64(c.Corporation.ID),
		Description:    c.Description,
		Gender:         c.Gender,
		ID:             int64(c.ID),
		Name:           c.Name,
		SecurityStatus: float64(c.SecurityStatus),
	}
	if c.Alliance.ID != 0 {
		arg.AllianceID.Int64 = int64(c.Alliance.ID)
		arg.AllianceID.Valid = true
	}
	if c.Faction.ID != 0 {
		arg.FactionID.Int64 = int64(c.Faction.ID)
		arg.FactionID.Valid = true
	}
	if c.SkillPoints != 0 {
		arg.SkillPoints.Int64 = int64(c.SkillPoints)
		arg.SkillPoints.Valid = true
	}
	if c.WalletBalance != 0 {
		arg.WalletBalance.Float64 = c.WalletBalance
		arg.WalletBalance.Valid = true
	}
	_, err := r.q.CreateCharacter(ctx, arg)
	if err != nil {
		if !isSqlite3ErrConstraint(err) {
			return err
		}
		arg := sqlc.UpdateCharacterParams{
			CorporationID:  int64(c.Corporation.ID),
			Description:    c.Description,
			ID:             int64(c.ID),
			Name:           c.Name,
			SecurityStatus: float64(c.SecurityStatus),
		}
		if c.Alliance.ID != 0 {
			arg.AllianceID.Int64 = int64(c.Alliance.ID)
			arg.AllianceID.Valid = true
		}
		if c.Faction.ID != 0 {
			arg.FactionID.Int64 = int64(c.Faction.ID)
			arg.FactionID.Valid = true
		}
		if c.SkillPoints != 0 {
			arg.SkillPoints.Int64 = int64(c.SkillPoints)
			arg.SkillPoints.Valid = true
		}
		if c.WalletBalance != 0 {
			arg.WalletBalance.Float64 = c.WalletBalance
			arg.WalletBalance.Valid = true
		}
		if err := r.q.UpdateCharacter(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}
