// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: eve_ship_skills.sql

package queries

import (
	"context"
)

const createShipSkill = `-- name: CreateShipSkill :exec
INSERT INTO eve_ship_skills (
    rank,
    ship_type_id,
    skill_type_id,
    skill_level
)
VALUES (
    ?, ?, ?, ?
)
`

type CreateShipSkillParams struct {
	Rank        int64
	ShipTypeID  int64
	SkillTypeID int64
	SkillLevel  int64
}

func (q *Queries) CreateShipSkill(ctx context.Context, arg CreateShipSkillParams) error {
	_, err := q.db.ExecContext(ctx, createShipSkill,
		arg.Rank,
		arg.ShipTypeID,
		arg.SkillTypeID,
		arg.SkillLevel,
	)
	return err
}

const getShipSkill = `-- name: GetShipSkill :one
SELECT
    rank,
    ship_type_id,
    skill_type_id,
    skt.name as skill_name,
    skill_level
FROM eve_ship_skills ess
JOIN eve_types as sht ON sht.id = ess.ship_type_id
JOIN eve_types as skt ON skt.id = ess.skill_type_id
WHERE ship_type_id = ? AND rank = ?
`

type GetShipSkillParams struct {
	ShipTypeID int64
	Rank       int64
}

type GetShipSkillRow struct {
	Rank        int64
	ShipTypeID  int64
	SkillTypeID int64
	SkillName   string
	SkillLevel  int64
}

func (q *Queries) GetShipSkill(ctx context.Context, arg GetShipSkillParams) (GetShipSkillRow, error) {
	row := q.db.QueryRowContext(ctx, getShipSkill, arg.ShipTypeID, arg.Rank)
	var i GetShipSkillRow
	err := row.Scan(
		&i.Rank,
		&i.ShipTypeID,
		&i.SkillTypeID,
		&i.SkillName,
		&i.SkillLevel,
	)
	return i, err
}

const listShipSkills = `-- name: ListShipSkills :many
SELECT
    rank,
    ship_type_id,
    skill_type_id,
    skt.name as skill_name,
    skill_level
FROM eve_ship_skills ess
JOIN eve_types as sht ON sht.id = ess.ship_type_id
JOIN eve_types as skt ON skt.id = ess.skill_type_id
WHERE ship_type_id = ?
ORDER BY RANK
`

type ListShipSkillsRow struct {
	Rank        int64
	ShipTypeID  int64
	SkillTypeID int64
	SkillName   string
	SkillLevel  int64
}

func (q *Queries) ListShipSkills(ctx context.Context, shipTypeID int64) ([]ListShipSkillsRow, error) {
	rows, err := q.db.QueryContext(ctx, listShipSkills, shipTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListShipSkillsRow
	for rows.Next() {
		var i ListShipSkillsRow
		if err := rows.Scan(
			&i.Rank,
			&i.ShipTypeID,
			&i.SkillTypeID,
			&i.SkillName,
			&i.SkillLevel,
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

const truncateShipSkills = `-- name: TruncateShipSkills :exec
DELETE FROM eve_ship_skills
`

func (q *Queries) TruncateShipSkills(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, truncateShipSkills)
	return err
}
