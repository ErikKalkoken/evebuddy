package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (st *Storage) GetEveShipSkill(ctx context.Context, shipTypeID int32, rank uint) (*model.EveShipSkill, error) {
	arg := queries.GetShipSkillParams{
		ShipTypeID: int64(shipTypeID),
		Rank:       int64(rank),
	}
	row, err := st.q.GetShipSkill(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get ShipSkill for %v: %w", arg, err)
	}
	return eveShipSkillFromDBModel(row.Rank, row.ShipTypeID, row.SkillLevel, row.SkillTypeID), nil
}

func (st *Storage) ListEveShipSkills(ctx context.Context, shipTypeID int32) ([]*model.EveShipSkill, error) {
	rows, err := st.q.ListShipSkills(ctx, int64(shipTypeID))
	if err != nil {
		return nil, fmt.Errorf("failed to list ship skills for ID %d: %w", shipTypeID, err)
	}
	oo := make([]*model.EveShipSkill, len(rows))
	for i, row := range rows {
		oo[i] = eveShipSkillFromDBModel(row.Rank, row.ShipTypeID, row.SkillLevel, row.SkillTypeID)
	}
	return oo, nil
}

func (st *Storage) UpdateEveShipSkills(ctx context.Context) error {
	if err := st.q.TruncateShipSkills(ctx); err != nil {
		return err
	}
	rows, err := st.listShipSkillsMap(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.PrimarySkillID.Valid && row.PrimarySkillLevel.Valid {
			if err := st.createShipSkillIfExists(ctx, 1, row.ShipTypeID, row.PrimarySkillID, row.PrimarySkillLevel); err != nil {
				return err
			}
			if err := st.createShipSkillIfExists(ctx, 2, row.ShipTypeID, row.SecondarySkillID, row.SecondarySkillLevel); err != nil {
				return err
			}
			if err := st.createShipSkillIfExists(ctx, 3, row.ShipTypeID, row.TertiarySkillID, row.TertiarySkillLevel); err != nil {
				return err
			}
			if err := st.createShipSkillIfExists(ctx, 4, row.ShipTypeID, row.QuaternarySkillID, row.QuaternarySkillLevel); err != nil {
				return err
			}
			if err := st.createShipSkillIfExists(ctx, 5, row.ShipTypeID, row.QuinarySkillID, row.QuinarySkillLevel); err != nil {
				return err
			}
			if err := st.createShipSkillIfExists(ctx, 6, row.ShipTypeID, row.SenarySkillID, row.SenarySkillLevel); err != nil {
				return err
			}
		}
	}
	return nil
}

func (st *Storage) createShipSkillIfExists(ctx context.Context, rank, skipTypeID int64, skillTypeID, level sql.NullInt64) error {
	if skillTypeID.Valid && level.Valid {
		arg := queries.CreateShipSkillParams{
			Rank:        rank,
			ShipTypeID:  skipTypeID,
			SkillTypeID: skillTypeID.Int64,
			SkillLevel:  level.Int64,
		}
		if err := st.q.CreateShipSkill(ctx, arg); err != nil {
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

func (st *Storage) CreateEveShipSkill(ctx context.Context, arg CreateShipSkillParams) error {
	if arg.ShipTypeID == 0 || arg.SkillTypeID == 0 || arg.SkillLevel == 0 {
		return fmt.Errorf("invalid arg %v", arg)
	}
	arg2 := queries.CreateShipSkillParams{
		Rank:        int64(arg.Rank),
		ShipTypeID:  int64(arg.ShipTypeID),
		SkillTypeID: int64(arg.SkillTypeID),
		SkillLevel:  int64(arg.SkillLevel),
	}
	if err := st.q.CreateShipSkill(ctx, arg2); err != nil {
		return fmt.Errorf("failed to create ShipSkill %v, %w", arg2, err)
	}
	return nil
}

func eveShipSkillFromDBModel(rank, shipTypeID, skillLevel, skillTypeID int64) *model.EveShipSkill {
	return &model.EveShipSkill{
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
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as primary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as primary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as secondary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as secondary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as tertiary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as tertiary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as quaternary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as quaternary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as quinary_skill_id,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as quinary_skill_level,
	(
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as senary_skill_id,
    (
		SELECT value
		FROM eve_type_dogma_attributes etda
		WHERE dogma_attribute_id = ?
		AND eve_type_id = et.id
	) as senary_skill_level
FROM eve_types et
JOIN eve_groups eg ON eg.id  = et.eve_group_id
WHERE eg.eve_category_id = ?
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

func (st *Storage) listShipSkillsMap(ctx context.Context) ([]listShipSkillsMapRow, error) {
	rows, err := st.db.QueryContext(
		ctx,
		listShipSkillsMapSQL,
		model.EveDogmaAttributeIDPrimarySkillID,
		model.EveDogmaAttributeIDPrimarySkillLevel,
		model.EveDogmaAttributeIDSecondarySkillID,
		model.EveDogmaAttributeIDSecondarySkillLevel,
		model.EveDogmaAttributeIDTertiarySkillID,
		model.EveDogmaAttributeIDTertiarySkillLevel,
		model.EveDogmaAttributeIDQuaternarySkillID,
		model.EveDogmaAttributeIDQuaternarySkillLevel,
		model.EveDogmaAttributeIDQuinarySkillID,
		model.EveDogmaAttributeIDQuinarySkillLevel,
		model.EveDogmaAttributeIDSenarySkillID,
		model.EveDogmaAttributeIDSenarySkillLevel,
		model.EveCategoryIDShip,
	)
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
