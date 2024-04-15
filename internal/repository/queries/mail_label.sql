-- name: CreateMailLabel :exec
INSERT INTO mail_labels (
    color,
    name,
    unread_count,
    character_id,
    label_id
)
VALUES (
    ?, ?, ?, ?, ?
);

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
    unread_count = ?
WHERE character_id = ?
AND label_id = ?;
