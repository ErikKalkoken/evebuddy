-- name: CreateCharacterSkillqueueItem :exec
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
    (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteCharacterSkillqueueItems :exec
DELETE FROM character_skillqueue_items
WHERE
    character_id = ?;

-- name: GetCharacterSkillqueueItem :one
SELECT
    sqlc.embed(csi),
    et.name as skill_name,
    et.description as skill_description,
    eg.name as group_name
FROM
    character_skillqueue_items csi
    JOIN eve_types et ON et.id = csi.eve_type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
WHERE
    character_id = ?
    and queue_position = ?;

-- name: ListCharacterSkillqueueItems :many
SELECT
    sqlc.embed(csi),
    et.name as skill_name,
    et.description as skill_description,
    eg.name as group_name
FROM
    character_skillqueue_items csi
    JOIN eve_types et ON et.id = csi.eve_type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
WHERE
    character_id = ?
ORDER BY
    queue_position;

-- name: GetTotalTrainingTime :one
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
    AND datetime(finish_date) > datetime();
