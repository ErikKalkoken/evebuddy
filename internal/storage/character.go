package storage

import (
	"context"
	"database/sql"
	"errors"
	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/queries"
	"fmt"
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
			Name:     row.CorporationName,
			Category: model.EveEntityCorporation,
		},
		Description: row.Description,
		Gender:      row.Gender,
		ID:          int32(row.ID),
		LastLoginAt: row.LastLoginAt,
		Name:        row.Name,
		Race: model.EveRace{
			ID:   int32(row.RaceID),
			Name: row.RaceName,
		},
		SecurityStatus: row.SecurityStatus,
		SkillPoints:    int(row.SkillPoints),
		Location: model.EveEntity{
			ID:       int32(row.LocationID),
			Name:     row.LocationName,
			Category: eveEntityCategoryFromDBModel(row.LocationCategory),
		},
		WalletBalance: row.WalletBalance,
	}
	c.Ship = eveTypeFromDBModel(row.EveType)
	c.Ship.Group = eveGroupFromDBModel(row.EveGroup)
	c.Ship.Group.Category = eveCategoryFromDBModel(row.EveCategory)
	if row.AllianceID.Valid {
		c.Alliance = model.EveEntity{
			ID:       int32(row.AllianceID.Int64),
			Name:     row.AllianceName.String,
			Category: model.EveEntityAlliance,
		}
	}
	if row.FactionID.Valid {
		c.Faction = model.EveEntity{
			ID:       int32(row.FactionID.Int64),
			Name:     row.FactionName.String,
			Category: model.EveEntityFaction,
		}
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
				Name:     row.CorporationName,
				Category: model.EveEntityCorporation,
			},
			Description:    row.Description,
			Gender:         row.Gender,
			ID:             int32(row.ID),
			LastLoginAt:    row.LastLoginAt,
			Name:           row.Name,
			Race:           model.EveRace{ID: int32(row.RaceID), Name: row.RaceName},
			SecurityStatus: row.SecurityStatus,
			SkillPoints:    int(row.SkillPoints),
			Location: model.EveEntity{
				ID:       int32(row.LocationID),
				Name:     row.LocationName,
				Category: eveEntityCategoryFromDBModel(row.LocationCategory),
			},
			// Ship: model.EveEntity{
			// 	ID:       int32(row.ShipID),
			// 	Name:     row.ShipName,
			// 	Category: model.EveEntityInventoryType,
			// },
			WalletBalance: row.WalletBalance,
		}
		if row.AllianceID.Valid {
			c.Alliance = model.EveEntity{
				ID:       int32(row.AllianceID.Int64),
				Name:     row.AllianceName.String,
				Category: model.EveEntityAlliance,
			}
		}
		if row.FactionID.Valid {
			c.Faction = model.EveEntity{
				ID:       int32(row.FactionID.Int64),
				Name:     row.FactionName.String,
				Category: model.EveEntityFaction,
			}
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
	arg := queries.UpdateOrCreateCharacterParams{
		Birthday:       c.Birthday,
		CorporationID:  int64(c.Corporation.ID),
		Description:    c.Description,
		Gender:         c.Gender,
		ID:             int64(c.ID),
		LastLoginAt:    c.LastLoginAt,
		Name:           c.Name,
		RaceID:         int64(c.Race.ID),
		SecurityStatus: float64(c.SecurityStatus),
		ShipID:         int64(c.Ship.ID),
		SkillPoints:    int64(c.SkillPoints),
		LocationID:     int64(c.Location.ID),
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
	_, err := r.q.UpdateOrCreateCharacter(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to update or create character %d: %w", c.ID, err)
	}
	return nil
}
