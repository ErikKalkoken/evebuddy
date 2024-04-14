-- name: CreateMailLabel :one
INSERT INTO mail_labels (
    character_id,
    color,
    label_id,
    name,
    unread_count
)
VALUES (
    ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetMailLabel :one
SELECT *
FROM mail_labels
WHERE character_id = ? AND label_id = ?;

-- name: ListMailLabels :many
SELECT *
FROM mail_labels
WHERE character_id = ?
AND label_id > 8
ORDER BY name;

-- name: ListMailLabelsByIDs :many
SELECT *
FROM mail_labels
WHERE character_id = ? AND label_id IN (sqlc.slice('ids'));

-- name: UpdateMailLabel :exec
UPDATE mail_labels
SET
    color = ?,
    name = ?,
    unread_count = ?;