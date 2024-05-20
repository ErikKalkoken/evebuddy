-- name: CreateCharacterSkillqueueItem :exec
INSERT INTO character_skillqueue_items (
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
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: DeleteCharacterSkillqueueItems :exec
DELETE FROM character_skillqueue_items
WHERE character_id = ?;

-- name: GetCharacterSkillqueueItem :one
SELECT sqlc.embed(character_skillqueue_items), eve_types.name as skill_name, eve_groups.name as group_name, eve_types.description as skill_description
FROM character_skillqueue_items
JOIN eve_types ON eve_types.id = character_skillqueue_items.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
WHERE character_id = ? and queue_position = ?;

-- name: ListCharacterSkillqueueItems :many
SELECT sqlc.embed(character_skillqueue_items), eve_types.name as skill_name, eve_groups.name as group_name, eve_types.description as skill_description
FROM character_skillqueue_items
JOIN eve_types ON eve_types.id = character_skillqueue_items.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
WHERE character_id = ?
AND start_date IS NOT NULL
AND finish_date IS NOT NULL
ORDER BY queue_position;

-- name: GetTotalTrainingTime :one
SELECT SUM(julianday(finish_date) - julianday(max(start_date, datetime())))
FROM character_skillqueue_items
WHERE character_id = ?
AND start_date IS NOT NULL
AND finish_date IS NOT NULL
AND datetime(finish_date) > datetime();
