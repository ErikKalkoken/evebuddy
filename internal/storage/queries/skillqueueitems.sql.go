// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: skillqueueitems.sql

package queries

import (
	"context"
	"database/sql"
)

const createSkillqueueItem = `-- name: CreateSkillqueueItem :exec
INSERT INTO skillqueue_items (
    eve_type_id,
    finish_date,
    finished_level,
    level_end_sp,
    level_start_sp,
    queue_position,
    my_character_id,
    start_date,
    training_start_sp
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
`

type CreateSkillqueueItemParams struct {
	EveTypeID       int64
	FinishDate      sql.NullTime
	FinishedLevel   int64
	LevelEndSp      sql.NullInt64
	LevelStartSp    sql.NullInt64
	QueuePosition   int64
	MyCharacterID   int64
	StartDate       sql.NullTime
	TrainingStartSp sql.NullInt64
}

func (q *Queries) CreateSkillqueueItem(ctx context.Context, arg CreateSkillqueueItemParams) error {
	_, err := q.db.ExecContext(ctx, createSkillqueueItem,
		arg.EveTypeID,
		arg.FinishDate,
		arg.FinishedLevel,
		arg.LevelEndSp,
		arg.LevelStartSp,
		arg.QueuePosition,
		arg.MyCharacterID,
		arg.StartDate,
		arg.TrainingStartSp,
	)
	return err
}

const deleteSkillqueueItems = `-- name: DeleteSkillqueueItems :exec
DELETE FROM skillqueue_items
WHERE my_character_id = ?
`

func (q *Queries) DeleteSkillqueueItems(ctx context.Context, myCharacterID int64) error {
	_, err := q.db.ExecContext(ctx, deleteSkillqueueItems, myCharacterID)
	return err
}

const getSkillqueueItem = `-- name: GetSkillqueueItem :one
SELECT skillqueue_items.eve_type_id, skillqueue_items.finish_date, skillqueue_items.finished_level, skillqueue_items.level_end_sp, skillqueue_items.level_start_sp, skillqueue_items.queue_position, skillqueue_items.my_character_id, skillqueue_items.start_date, skillqueue_items.training_start_sp, eve_types.name as skill_name, eve_groups.name as group_name
FROM skillqueue_items
JOIN eve_types ON eve_types.id = skillqueue_items.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
WHERE my_character_id = ? and queue_position = ?
`

type GetSkillqueueItemParams struct {
	MyCharacterID int64
	QueuePosition int64
}

type GetSkillqueueItemRow struct {
	SkillqueueItem SkillqueueItem
	SkillName      string
	GroupName      string
}

func (q *Queries) GetSkillqueueItem(ctx context.Context, arg GetSkillqueueItemParams) (GetSkillqueueItemRow, error) {
	row := q.db.QueryRowContext(ctx, getSkillqueueItem, arg.MyCharacterID, arg.QueuePosition)
	var i GetSkillqueueItemRow
	err := row.Scan(
		&i.SkillqueueItem.EveTypeID,
		&i.SkillqueueItem.FinishDate,
		&i.SkillqueueItem.FinishedLevel,
		&i.SkillqueueItem.LevelEndSp,
		&i.SkillqueueItem.LevelStartSp,
		&i.SkillqueueItem.QueuePosition,
		&i.SkillqueueItem.MyCharacterID,
		&i.SkillqueueItem.StartDate,
		&i.SkillqueueItem.TrainingStartSp,
		&i.SkillName,
		&i.GroupName,
	)
	return i, err
}

const listSkillqueueItems = `-- name: ListSkillqueueItems :many
SELECT skillqueue_items.eve_type_id, skillqueue_items.finish_date, skillqueue_items.finished_level, skillqueue_items.level_end_sp, skillqueue_items.level_start_sp, skillqueue_items.queue_position, skillqueue_items.my_character_id, skillqueue_items.start_date, skillqueue_items.training_start_sp, eve_types.name as skill_name, eve_groups.name as group_name
FROM skillqueue_items
JOIN eve_types ON eve_types.id = skillqueue_items.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
WHERE my_character_id = ?
ORDER BY queue_position
`

type ListSkillqueueItemsRow struct {
	SkillqueueItem SkillqueueItem
	SkillName      string
	GroupName      string
}

func (q *Queries) ListSkillqueueItems(ctx context.Context, myCharacterID int64) ([]ListSkillqueueItemsRow, error) {
	rows, err := q.db.QueryContext(ctx, listSkillqueueItems, myCharacterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListSkillqueueItemsRow
	for rows.Next() {
		var i ListSkillqueueItemsRow
		if err := rows.Scan(
			&i.SkillqueueItem.EveTypeID,
			&i.SkillqueueItem.FinishDate,
			&i.SkillqueueItem.FinishedLevel,
			&i.SkillqueueItem.LevelEndSp,
			&i.SkillqueueItem.LevelStartSp,
			&i.SkillqueueItem.QueuePosition,
			&i.SkillqueueItem.MyCharacterID,
			&i.SkillqueueItem.StartDate,
			&i.SkillqueueItem.TrainingStartSp,
			&i.SkillName,
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