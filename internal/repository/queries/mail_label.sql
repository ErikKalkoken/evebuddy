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

-- name: UpdateOrCreateMailLabel :exec
INSERT INTO mail_labels (
    color,
    name,
    unread_count,
    character_id,
    label_id
)
VALUES (
    ?, ?, ?, ?, ?
)
ON CONFLICT (character_id, label, id) DO
UPDATE
SET
    color = ?,
    name = ?,
    unread_count = ?;
