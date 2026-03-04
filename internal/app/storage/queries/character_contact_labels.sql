-- name: UpdateOrCreateCharacterContactLabel :exec
INSERT INTO
    character_contact_labels (character_id, label_id, name)
VALUES
    (?1, ?2, ?3)
ON CONFLICT (character_id, label_id) DO UPDATE
SET
    name = ?3;

-- name: DeleteCharacterContactLabels :exec
DELETE FROM character_contact_labels
WHERE
    character_id = ?
    AND label_id IN (sqlc.slice('label_ids'));

-- name: GetCharacterContactLabel :one
SELECT
    *
FROM
    character_contact_labels
WHERE
    character_id = ?
    AND label_id = ?;

-- name: ListCharacterContactLabels :many
SELECT
    *
FROM
    character_contact_labels
WHERE
    character_id = ?;

-- name: ListCharacterContactLabelIDs :many
SELECT
    label_id
FROM
    character_contact_labels
WHERE
    character_id = ?;
