package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (r *Storage) GetShipSkill(ctx context.Context, shipTypeID int32, rank uint) (*model.ShipSkill, error) {
	arg := queries.GetShipSkillParams{
		ShipTypeID: int64(shipTypeID),
		Rank:       int64(rank),
	}
	row, err := r.q.GetShipSkill(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get ShipSkill for %v: %w", arg, err)
	}
	return shipSkillFromDBModel(row.Rank, row.ShipTypeID, row.SkillLevel, row.SkillTypeID), nil
}

func (r *Storage) ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*model.CharacterShipAbility, error) {
	arg := queries.ListCharacterShipsAbilitiesParams{
		CharacterID: int64(characterID),
		Name:        search,
	}
	rows, err := r.q.ListCharacterShipsAbilities(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list ship abilities for character %d and search %s: %w", characterID, search, err)
	}
	oo := make([]*model.CharacterShipAbility, len(rows))
	for i, row := range rows {
		o := &model.CharacterShipAbility{
			Group:  model.EntityShort[int32]{ID: int32(row.GroupID), Name: row.GroupName},
			Type:   model.EntityShort[int32]{ID: int32(row.TypeID), Name: row.TypeName},
			CanFly: row.CanFly,
		}
		oo[i] = o
	}
	return oo, nil
}

func (r *Storage) ListShipSkills(ctx context.Context, shipTypeID int32) ([]*model.ShipSkill, error) {
	rows, err := r.q.ListShipSkills(ctx, int64(shipTypeID))
	if err != nil {
		return nil, fmt.Errorf("failed to list ship skills for ID %d: %w", shipTypeID, err)
	}
	oo := make([]*model.ShipSkill, len(rows))
	for i, row := range rows {
		oo[i] = shipSkillFromDBModel(row.Rank, row.ShipTypeID, row.SkillLevel, row.SkillTypeID)
	}
	return oo, nil
}

func (r *Storage) UpdateShipSkills(ctx context.Context) error {
	if err := r.q.TruncateShipSkills(ctx); err != nil {
		return err
	}
	rows, err := r.listShipSkillsMap(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.PrimarySkillID.Valid && row.PrimarySkillLevel.Valid {
			if err := r.createShipSkillIfExists(ctx, 1, row.ShipTypeID, row.PrimarySkillID, row.PrimarySkillLevel); err != nil {
				return err
			}
			if err := r.createShipSkillIfExists(ctx, 2, row.ShipTypeID, row.SecondarySkillID, row.SecondarySkillLevel); err != nil {
				return err
			}
			if err := r.createShipSkillIfExists(ctx, 3, row.ShipTypeID, row.TertiarySkillID, row.TertiarySkillLevel); err != nil {
				return err
			}
			if err := r.createShipSkillIfExists(ctx, 4, row.ShipTypeID, row.QuaternarySkillID, row.QuaternarySkillLevel); err != nil {
				return err
			}
			if err := r.createShipSkillIfExists(ctx, 5, row.ShipTypeID, row.QuinarySkillID, row.QuaternarySkillLevel); err != nil {
				return err
			}
			if err := r.createShipSkillIfExists(ctx, 6, row.ShipTypeID, row.SenarySkillID, row.SenarySkillLevel); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Storage) createShipSkillIfExists(ctx context.Context, rank, skipTypeID int64, skillTypeID, level sql.NullInt64) error {
	if skillTypeID.Valid && level.Valid {
		arg := queries.CreateShipSkillParams{
			Rank:        rank,
			ShipTypeID:  skipTypeID,
			SkillTypeID: skillTypeID.Int64,
			SkillLevel:  level.Int64,
		}
		if err := r.q.CreateShipSkill(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

type CreateShipSkillParams struct {
	Rank        uint
	ShipTypeID  int32
	SkillTypeID int32
	SkillLevel  uint
}

func (r *Storage) CreateShipSkill(ctx context.Context, arg CreateShipSkillParams) error {
	if arg.ShipTypeID == 0 || arg.SkillTypeID == 0 || arg.SkillLevel == 0 {
		return fmt.Errorf("invalid arg %v", arg)
	}
	arg2 := queries.CreateShipSkillParams{
		Rank:        int64(arg.Rank),
		ShipTypeID:  int64(arg.ShipTypeID),
		SkillTypeID: int64(arg.SkillTypeID),
		SkillLevel:  int64(arg.SkillLevel),
	}
	if err := r.q.CreateShipSkill(ctx, arg2); err != nil {
		return fmt.Errorf("failed to create ShipSkill %v, %w", arg2, err)
	}
	return nil
}

func shipSkillFromDBModel(rank, shipTypeID, skillLevel, skillTypeID int64) *model.ShipSkill {
	return &model.ShipSkill{
		Rank:        uint(rank),
		ShipTypeID:  int32(shipTypeID),
		SkillTypeID: int32(skillTypeID),
		SkillLevel:  uint(skillLevel),
	}
}

const listShipSkillsMapSQL = `-- name: ListShipSkillsMap :many
SELECT
	et.id as ship_type_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 182
		AND eve_type_id = et.id
	) as primary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 277
		AND eve_type_id = et.id
	) as primary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 183
		AND eve_type_id = et.id
	) as secondary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 278
		AND eve_type_id = et.id
	) as secondary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 184
		AND eve_type_id = et.id
	) as tertiary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 279
		AND eve_type_id = et.id
	) as tertiary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 1285
		AND eve_type_id = et.id
	) as quaternary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 1286
		AND eve_type_id = et.id
	) as quaternary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 1289
		AND eve_type_id = et.id
	) as quinary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 1287
		AND eve_type_id = et.id
	) as quinary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 1290
		AND eve_type_id = et.id
	) as senary_skill_id,
    (
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = 1288
		AND eve_type_id = et.id
	) as senary_skill_level
FROM eve_types et
JOIN eve_groups eg ON eg.id  = et.eve_group_id
WHERE eg.eve_category_id = 6
AND et.is_published IS TRUE
`

type listShipSkillsMapRow struct {
	ShipTypeID           int64
	PrimarySkillID       sql.NullInt64
	PrimarySkillLevel    sql.NullInt64
	SecondarySkillID     sql.NullInt64
	SecondarySkillLevel  sql.NullInt64
	TertiarySkillID      sql.NullInt64
	TertiarySkillLevel   sql.NullInt64
	QuaternarySkillID    sql.NullInt64
	QuaternarySkillLevel sql.NullInt64
	QuinarySkillID       sql.NullInt64
	QuinarySkillLevel    sql.NullInt64
	SenarySkillID        sql.NullInt64
	SenarySkillLevel     sql.NullInt64
}

func (r *Storage) listShipSkillsMap(ctx context.Context) ([]listShipSkillsMapRow, error) {
	rows, err := r.db.QueryContext(ctx, listShipSkillsMapSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []listShipSkillsMapRow
	for rows.Next() {
		var i listShipSkillsMapRow
		if err := rows.Scan(
			&i.ShipTypeID,
			&i.PrimarySkillID,
			&i.PrimarySkillLevel,
			&i.SecondarySkillID,
			&i.SecondarySkillLevel,
			&i.TertiarySkillID,
			&i.TertiarySkillLevel,
			&i.QuaternarySkillID,
			&i.QuaternarySkillLevel,
			&i.QuinarySkillID,
			&i.QuinarySkillLevel,
			&i.SenarySkillID,
			&i.SenarySkillLevel,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
