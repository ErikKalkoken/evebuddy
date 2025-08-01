// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: character_skillqueue_items.sql

package queries

import (
	"context"
	"database/sql"
)

const createCharacterSkillqueueItem = `-- name: CreateCharacterSkillqueueItem :exec
INSERT INTO
    character_skillqueue_items (
        eve_type_id,
        finish_date,
        finished_level,
        level_end_sp,
        level_start_sp,
        queue_position,
        character_id,
        start_date,
        training_start_sp
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?)
`

type CreateCharacterSkillqueueItemParams struct {
	EveTypeID       int64
	FinishDate      sql.NullTime
	FinishedLevel   int64
	LevelEndSp      sql.NullInt64
	LevelStartSp    sql.NullInt64
	QueuePosition   int64
	CharacterID     int64
	StartDate       sql.NullTime
	TrainingStartSp sql.NullInt64
}

func (q *Queries) CreateCharacterSkillqueueItem(ctx context.Context, arg CreateCharacterSkillqueueItemParams) error {
	_, err := q.db.ExecContext(ctx, createCharacterSkillqueueItem,
		arg.EveTypeID,
		arg.FinishDate,
		arg.FinishedLevel,
		arg.LevelEndSp,
		arg.LevelStartSp,
		arg.QueuePosition,
		arg.CharacterID,
		arg.StartDate,
		arg.TrainingStartSp,
	)
	return err
}

const deleteCharacterSkillqueueItems = `-- name: DeleteCharacterSkillqueueItems :exec
DELETE FROM character_skillqueue_items
WHERE
    character_id = ?
`

func (q *Queries) DeleteCharacterSkillqueueItems(ctx context.Context, characterID int64) error {
	_, err := q.db.ExecContext(ctx, deleteCharacterSkillqueueItems, characterID)
	return err
}

const getCharacterSkillqueueItem = `-- name: GetCharacterSkillqueueItem :one
SELECT
    csi.id, csi.character_id, csi.eve_type_id, csi.finish_date, csi.finished_level, csi.level_end_sp, csi.level_start_sp, csi.queue_position, csi.start_date, csi.training_start_sp,
    et.name as skill_name,
    et.description as skill_description,
    eg.name as group_name
FROM
    character_skillqueue_items csi
    JOIN eve_types et ON et.id = csi.eve_type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
WHERE
    character_id = ?
    and queue_position = ?
`

type GetCharacterSkillqueueItemParams struct {
	CharacterID   int64
	QueuePosition int64
}

type GetCharacterSkillqueueItemRow struct {
	CharacterSkillqueueItem CharacterSkillqueueItem
	SkillName               string
	SkillDescription        string
	GroupName               string
}

func (q *Queries) GetCharacterSkillqueueItem(ctx context.Context, arg GetCharacterSkillqueueItemParams) (GetCharacterSkillqueueItemRow, error) {
	row := q.db.QueryRowContext(ctx, getCharacterSkillqueueItem, arg.CharacterID, arg.QueuePosition)
	var i GetCharacterSkillqueueItemRow
	err := row.Scan(
		&i.CharacterSkillqueueItem.ID,
		&i.CharacterSkillqueueItem.CharacterID,
		&i.CharacterSkillqueueItem.EveTypeID,
		&i.CharacterSkillqueueItem.FinishDate,
		&i.CharacterSkillqueueItem.FinishedLevel,
		&i.CharacterSkillqueueItem.LevelEndSp,
		&i.CharacterSkillqueueItem.LevelStartSp,
		&i.CharacterSkillqueueItem.QueuePosition,
		&i.CharacterSkillqueueItem.StartDate,
		&i.CharacterSkillqueueItem.TrainingStartSp,
		&i.SkillName,
		&i.SkillDescription,
		&i.GroupName,
	)
	return i, err
}

const getTotalTrainingTime = `-- name: GetTotalTrainingTime :one
SELECT
    SUM(
        julianday(finish_date) - julianday(max(start_date, datetime()))
    )
FROM
    character_skillqueue_items
WHERE
    character_id = ?
    AND start_date IS NOT NULL
    AND finish_date IS NOT NULL
    AND datetime(finish_date) > datetime()
`

func (q *Queries) GetTotalTrainingTime(ctx context.Context, characterID int64) (sql.NullFloat64, error) {
	row := q.db.QueryRowContext(ctx, getTotalTrainingTime, characterID)
	var sum sql.NullFloat64
	err := row.Scan(&sum)
	return sum, err
}

const listCharacterSkillqueueItems = `-- name: ListCharacterSkillqueueItems :many
SELECT
    csi.id, csi.character_id, csi.eve_type_id, csi.finish_date, csi.finished_level, csi.level_end_sp, csi.level_start_sp, csi.queue_position, csi.start_date, csi.training_start_sp,
    et.name as skill_name,
    et.description as skill_description,
    eg.name as group_name
FROM
    character_skillqueue_items csi
    JOIN eve_types et ON et.id = csi.eve_type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
WHERE
    character_id = ?
    AND start_date IS NOT NULL
    AND finish_date IS NOT NULL
ORDER BY
    queue_position
`

type ListCharacterSkillqueueItemsRow struct {
	CharacterSkillqueueItem CharacterSkillqueueItem
	SkillName               string
	SkillDescription        string
	GroupName               string
}

func (q *Queries) ListCharacterSkillqueueItems(ctx context.Context, characterID int64) ([]ListCharacterSkillqueueItemsRow, error) {
	rows, err := q.db.QueryContext(ctx, listCharacterSkillqueueItems, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListCharacterSkillqueueItemsRow
	for rows.Next() {
		var i ListCharacterSkillqueueItemsRow
		if err := rows.Scan(
			&i.CharacterSkillqueueItem.ID,
			&i.CharacterSkillqueueItem.CharacterID,
			&i.CharacterSkillqueueItem.EveTypeID,
			&i.CharacterSkillqueueItem.FinishDate,
			&i.CharacterSkillqueueItem.FinishedLevel,
			&i.CharacterSkillqueueItem.LevelEndSp,
			&i.CharacterSkillqueueItem.LevelStartSp,
			&i.CharacterSkillqueueItem.QueuePosition,
			&i.CharacterSkillqueueItem.StartDate,
			&i.CharacterSkillqueueItem.TrainingStartSp,
			&i.SkillName,
			&i.SkillDescription,
			&i.GroupName,
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
