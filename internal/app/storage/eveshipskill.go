package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetEveShipSkill(ctx context.Context, shipTypeID int32, rank uint) (*app.EveShipSkill, error) {
	arg := queries.GetShipSkillParams{
		ShipTypeID: int64(shipTypeID),
		Rank:       int64(rank),
	}
	row, err := st.qRO.GetShipSkill(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get ShipSkill for %+v: %w", arg, convertGetError(err))
	}
	return eveShipSkillFromDBModel(row.Rank, row.ShipTypeID, row.SkillTypeID, row.SkillName, row.SkillLevel), nil
}

func (st *Storage) ListEveShipSkills(ctx context.Context, shipTypeID int32) ([]*app.EveShipSkill, error) {
	rows, err := st.qRO.ListShipSkills(ctx, int64(shipTypeID))
	if err != nil {
		return nil, fmt.Errorf("list ship skills for ID %d: %w", shipTypeID, err)
	}
	oo := make([]*app.EveShipSkill, len(rows))
	for i, row := range rows {
		oo[i] = eveShipSkillFromDBModel(row.Rank, row.ShipTypeID, row.SkillTypeID, row.SkillName, row.SkillLevel)
	}
	return oo, nil
}

func (st *Storage) UpdateEveShipSkills(ctx context.Context) error {
	rows, err := st.listShipSkillsMap(ctx)
	if err != nil {
		return err
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	if err := qtx.TruncateShipSkills(ctx); err != nil {
		return err
	}
	for _, row := range rows {
		if row.PrimarySkillID.Valid && row.PrimarySkillLevel.Valid {
			for rank := int64(1); rank <= 6; rank++ {
				var skillID, skillLevel sql.NullInt64
				switch rank {
				case 1:
					skillID = row.PrimarySkillID
					skillLevel = row.PrimarySkillLevel
				case 2:
					skillID = row.SecondarySkillID
					skillLevel = row.SecondarySkillLevel
				case 3:
					skillID = row.TertiarySkillID
					skillLevel = row.TertiarySkillLevel
				case 4:
					skillID = row.QuaternarySkillID
					skillLevel = row.QuaternarySkillLevel
				case 5:
					skillID = row.QuinarySkillID
					skillLevel = row.QuinarySkillLevel
				case 6:
					skillID = row.SenarySkillID
					skillLevel = row.SenarySkillLevel
				}
				if err := st.createShipSkillIfExists(ctx, qtx, rank, row.ShipTypeID, skillID, skillLevel); err != nil {
					return err
				}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (st *Storage) createShipSkillIfExists(ctx context.Context, q *queries.Queries, rank, skipTypeID int64, skillTypeID, level sql.NullInt64) error {
	if skillTypeID.Valid && level.Valid {
		arg := queries.CreateShipSkillParams{
			Rank:        rank,
			ShipTypeID:  skipTypeID,
			SkillTypeID: skillTypeID.Int64,
			SkillLevel:  level.Int64,
		}
		if err := q.CreateShipSkill(ctx, arg); err != nil {
			return fmt.Errorf("create ship skill: %+v, %w", arg, err)
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
	wrapErr := func(err error) error {
		return fmt.Errorf("createEveShipSkill: %+v: %w", arg, err)
	}
	if arg.ShipTypeID == 0 || arg.SkillTypeID == 0 || arg.SkillLevel == 0 {
		return wrapErr(app.ErrInvalid)
	}
	arg2 := queries.CreateShipSkillParams{
		Rank:        int64(arg.Rank),
		ShipTypeID:  int64(arg.ShipTypeID),
		SkillTypeID: int64(arg.SkillTypeID),
		SkillLevel:  int64(arg.SkillLevel),
	}
	if err := st.qRW.CreateShipSkill(ctx, arg2); err != nil {
		return wrapErr(err)
	}
	return nil
}

func eveShipSkillFromDBModel(rank, shipTypeID, skillTypeID int64, skillName string, skillLevel int64) *app.EveShipSkill {
	return &app.EveShipSkill{
		Rank:        uint(rank),
		ShipTypeID:  int32(shipTypeID),
		SkillTypeID: int32(skillTypeID),
		SkillName:   skillName,
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
	rows, err := st.dbRO.QueryContext(
		ctx,
		listShipSkillsMapSQL,
		app.EveDogmaAttributePrimarySkillID,
		app.EveDogmaAttributePrimarySkillLevel,
		app.EveDogmaAttributeSecondarySkillID,
		app.EveDogmaAttributeSecondarySkillLevel,
		app.EveDogmaAttributeTertiarySkillID,
		app.EveDogmaAttributeTertiarySkillLevel,
		app.EveDogmaAttributeQuaternarySkillID,
		app.EveDogmaAttributeQuaternarySkillLevel,
		app.EveDogmaAttributeQuinarySkillID,
		app.EveDogmaAttributeQuinarySkillLevel,
		app.EveDogmaAttributeSenarySkillID,
		app.EveDogmaAttributeSenarySkillLevel,
		app.EveCategoryShip,
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
