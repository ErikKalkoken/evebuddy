-- name: CreateMailLabel :one
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
RETURNING *;

-- name: DeleteObsoleteMailLabels :exec
DELETE FROM mail_labels
WHERE mail_labels.character_id = ?
AND id NOT IN (
    SELECT mail_label_id
    FROM mail_mail_labels
    JOIN mails ON mails.id = mail_mail_labels.mail_id
    WHERE mail_labels.character_id = ?
);

-- name: GetMailLabel :one
SELECT *
FROM mail_labels
WHERE character_id = ? AND label_id = ?;

-- name: ListMailLabelsOrdered :many
SELECT *
FROM mail_labels
WHERE character_id = ?
AND label_id > 8
ORDER BY name;

-- name: ListMailLabelsByIDs :many
SELECT *
FROM mail_labels
WHERE character_id = ? AND label_id IN (sqlc.slice('ids'));

-- name: UpdateOrCreateMailLabel :one
INSERT INTO mail_labels (
    character_id,
    label_id,
    color,
    name,
    unread_count
)
VALUES (
    ?1, ?2, ?3, ?4, ?5
)
ON CONFLICT(character_id, label_id) DO
UPDATE SET
    color = ?3,
    name = ?4,
    unread_count = ?5
WHERE character_id = ?1
AND label_id = ?2
RETURNING *;
