-- name: CreateSkillqueueItem :exec
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
);

-- name: DeleteSkillqueueItems :exec
DELETE FROM skillqueue_items
WHERE my_character_id = ?;

-- name: GetSkillqueueItem :one
SELECT sqlc.embed(skillqueue_items), eve_types.name as skill_name, eve_groups.name as group_name, eve_types.description as skill_description
FROM skillqueue_items
JOIN eve_types ON eve_types.id = skillqueue_items.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
WHERE my_character_id = ? and queue_position = ?;

-- name: ListSkillqueueItems :many
SELECT sqlc.embed(skillqueue_items), eve_types.name as skill_name, eve_groups.name as group_name, eve_types.description as skill_description
FROM skillqueue_items
JOIN eve_types ON eve_types.id = skillqueue_items.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
WHERE my_character_id = ?
ORDER BY queue_position;
