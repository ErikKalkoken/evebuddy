-- name: CreateCharacterMailLabel :one
INSERT INTO
    character_mail_labels (color, name, unread_count, character_id, label_id)
VALUES
    (?, ?, ?, ?, ?) RETURNING *;

-- name: DeleteObsoleteCharacterMailLabels :exec
DELETE FROM character_mail_labels
WHERE
    character_mail_labels.character_id = ?
    AND id NOT IN (
        SELECT
            character_mail_label_id
        FROM
            character_mail_mail_labels
            JOIN character_mails ON character_mails.id = character_mail_mail_labels.character_mail_id
        WHERE
            character_mail_labels.character_id = ?
    );

-- name: GetCharacterMailLabel :one
SELECT
    *
FROM
    character_mail_labels
WHERE
    character_id = ?
    AND label_id = ?;

-- name: ListCharacterMailLabelsOrdered :many
SELECT
    *
FROM
    character_mail_labels
WHERE
    character_id = ?
    AND label_id > 8
ORDER BY
    name;

-- name: ListCharacterMailLabelsByIDs :many
SELECT
    *
FROM
    character_mail_labels
WHERE
    character_id = ?
    AND label_id IN (sqlc.slice ('ids'));

-- name: UpdateOrCreateCharacterMailLabel :one
INSERT INTO
    character_mail_labels (character_id, label_id, color, name, unread_count)
VALUES
    (?1, ?2, ?3, ?4, ?5)
ON CONFLICT (character_id, label_id) DO UPDATE
SET
    color = ?3,
    name = ?4,
    unread_count = ?5 RETURNING *;
