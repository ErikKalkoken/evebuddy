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
SELECT *
FROM skillqueue_items
WHERE my_character_id = ? and eve_type_id = ?;

-- name: ListSkillqueueItems :many
SELECT *
FROM skillqueue_items
WHERE my_character_id = ?;
