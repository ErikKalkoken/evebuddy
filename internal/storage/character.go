package storage

import (
	"context"
	"database/sql"
	"errors"
	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/sqlc"
	"fmt"

	"github.com/mattn/go-sqlite3"
)

func (r *Storage) DeleteCharacter(ctx context.Context, characterID int32) error {
	err := r.q.DeleteCharacter(ctx, int64(characterID))
	if err != nil {
		return fmt.Errorf("failed to delete character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) GetCharacter(ctx context.Context, characterID int32) (model.Character, error) {
	row, err := r.q.GetCharacter(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return model.Character{}, fmt.Errorf("failed to get character %d: %w", characterID, err)
	}
	c := model.Character{
		Birthday: row.Birthday,
		Corporation: model.EveEntity{ID: int32(row.CorporationID),
			Name:     row.Name_2,
			Category: eveEntityCategoryFromDBModel(row.Category),
		},
		Description: row.Description,
		Gender:      row.Gender,
		ID:          int32(row.ID),
		LastLoginAt: row.LastLoginAt,
		Name:        row.Name,
		Race: model.Race{
			ID:   int32(row.RaceID),
			Name: row.RaceName,
		},
		SecurityStatus: row.SecurityStatus,
		SkillPoints:    int(row.SkillPoints),
		SolarSystem: model.EveEntity{
			ID:       int32(row.SolarSystemID),
			Name:     row.Name_5,
			Category: eveEntityCategoryFromDBModel(row.Category_4),
		},
		WalletBalance: row.WalletBalance,
	}
	if row.AllianceID.Valid {
		c.Alliance = model.EveEntity{
			ID:       int32(row.AllianceID.Int64),
			Name:     row.Name_3.String,
			Category: eveEntityCategoryFromDBModel(row.Category_2.String),
		}
	}
	if row.FactionID.Valid {
		c.Faction = model.EveEntity{
			ID:       int32(row.FactionID.Int64),
			Name:     row.Name_4.String,
			Category: eveEntityCategoryFromDBModel(row.Category_3.String),
		}
	}
	if row.MailUpdatedAt.Valid {
		c.MailUpdatedAt = row.MailUpdatedAt.Time
	}
	return c, nil
}

func (r *Storage) GetFirstCharacter(ctx context.Context) (model.Character, error) {
	ids, err := r.ListCharacterIDs(ctx)
	if err != nil {
		return model.Character{}, nil
	}
	if len(ids) == 0 {
		return model.Character{}, ErrNotFound
	}
	return r.GetCharacter(ctx, ids[0])

}

func (r *Storage) ListCharacters(ctx context.Context) ([]model.Character, error) {
	rows, err := r.q.ListCharacters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list characters: %w", err)

	}
	cc := make([]model.Character, len(rows))
	for i, row := range rows {
		c := model.Character{
			Birthday: row.Birthday,
			Corporation: model.EveEntity{ID: int32(row.CorporationID),
				Name:     row.Name_2,
				Category: eveEntityCategoryFromDBModel(row.Category),
			},
			Description:    row.Description,
			Gender:         row.Gender,
			ID:             int32(row.ID),
			LastLoginAt:    row.LastLoginAt,
			Name:           row.Name,
			Race:           model.Race{ID: int32(row.RaceID), Name: row.RaceName},
			SecurityStatus: row.SecurityStatus,
			SkillPoints:    int(row.SkillPoints),
			SolarSystem: model.EveEntity{
				ID:       int32(row.SolarSystemID),
				Name:     row.Name_5,
				Category: eveEntityCategoryFromDBModel(row.Category_4),
			},
			WalletBalance: row.WalletBalance,
		}
		if row.AllianceID.Valid {
			c.Alliance = model.EveEntity{
				ID:       int32(row.AllianceID.Int64),
				Name:     row.Name_3.String,
				Category: eveEntityCategoryFromDBModel(row.Category_2.String),
			}
		}
		if row.FactionID.Valid {
			c.Faction = model.EveEntity{
				ID:       int32(row.FactionID.Int64),
				Name:     row.Name_4.String,
				Category: eveEntityCategoryFromDBModel(row.Category_3.String),
			}
		}
		if row.MailUpdatedAt.Valid {
			c.MailUpdatedAt = row.MailUpdatedAt.Time
		}
		cc[i] = c
	}
	return cc, nil
}

func (r *Storage) ListCharacterIDs(ctx context.Context) ([]int32, error) {
	ids, err := r.q.ListCharacterIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list character IDs: %w", err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Storage) UpdateOrCreateCharacter(ctx context.Context, c *model.Character) error {
	if c.Corporation.ID == 0 {
		return fmt.Errorf("can not store character without a corporation: %d", c.ID)
	}
	if c.SolarSystem.ID == 0 {
		return fmt.Errorf("can not store character without a solar system: %d", c.ID)
	}
	if c.Race.ID == 0 {
		return fmt.Errorf("can not store character without a race: %d", c.ID)
	}
	err := func() error {
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		qtx := r.q.WithTx(tx)
		arg := sqlc.CreateCharacterParams{
			Birthday:       c.Birthday,
			CorporationID:  int64(c.Corporation.ID),
			Description:    c.Description,
			Gender:         c.Gender,
			ID:             int64(c.ID),
			LastLoginAt:    c.LastLoginAt,
			Name:           c.Name,
			RaceID:         int64(c.Race.ID),
			SecurityStatus: float64(c.SecurityStatus),
			SkillPoints:    int64(c.SkillPoints),
			SolarSystemID:  int64(c.SolarSystem.ID),
			WalletBalance:  c.WalletBalance,
		}
		if c.Alliance.ID != 0 {
			arg.AllianceID.Int64 = int64(c.Alliance.ID)
			arg.AllianceID.Valid = true
		}
		if c.Faction.ID != 0 {
			arg.FactionID.Int64 = int64(c.Faction.ID)
			arg.FactionID.Valid = true
		}
		_, err = qtx.CreateCharacter(ctx, arg)
		if err != nil {
			sqlErr, ok := err.(sqlite3.Error)
			if !ok || sqlErr.ExtendedCode != sqlite3.ErrConstraintPrimaryKey {
				return err
			}
			arg := sqlc.UpdateCharacterParams{
				CorporationID:  int64(c.Corporation.ID),
				Description:    c.Description,
				ID:             int64(c.ID),
				LastLoginAt:    c.LastLoginAt,
				Name:           c.Name,
				SecurityStatus: float64(c.SecurityStatus),
				SkillPoints:    int64(c.SkillPoints),
				SolarSystemID:  int64(c.SolarSystem.ID),
				WalletBalance:  c.WalletBalance,
			}
			if c.Alliance.ID != 0 {
				arg.AllianceID.Int64 = int64(c.Alliance.ID)
				arg.AllianceID.Valid = true
			}
			if c.Faction.ID != 0 {
				arg.FactionID.Int64 = int64(c.Faction.ID)
				arg.FactionID.Valid = true
			}
			if err := qtx.UpdateCharacter(ctx, arg); err != nil {
				return err
			}
		}
		return tx.Commit()
	}()
	if err != nil {
		return fmt.Errorf("failed to update or create character %d: %w", c.ID, err)
	}
	return nil
}
