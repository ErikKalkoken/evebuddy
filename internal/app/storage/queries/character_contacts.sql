-- name: DeleteCharacterContacts :exec
DELETE FROM character_contacts
WHERE
    character_id = ?
    AND contact_id IN (sqlc.slice('contact_ids'));

-- name: GetCharacterContact :one
SELECT
    sqlc.embed(cc),
    sqlc.embed(ee)
FROM
    character_contacts cc
    JOIN eve_entities as ee ON ee.id = cc.contact_id
WHERE
    character_id = ?
    AND contact_id = ?;

-- name: ListCharacterContactIDs :many
SELECT
    contact_id
FROM
    character_contacts
WHERE
    character_id = ?;

-- name: ListCharacterContacts :many
SELECT
    sqlc.embed(cc),
    sqlc.embed(ee)
FROM
    character_contacts cc
    JOIN eve_entities as ee ON ee.id = cc.contact_id
WHERE
    character_id = ?;

-- name: UpdateOrCreateCharacterContact :exec
INSERT INTO
    character_contacts (
        character_id,
        contact_id,
        is_blocked,
        is_watched,
        standing
    )
VALUES
    (?1, ?2, ?3, ?4, ?5)
ON CONFLICT (character_id, contact_id) DO UPDATE
SET
    is_blocked = ?3,
    is_watched = ?4,
    standing = ?5;

-- name: CreateCharacterContactLabel :exec
INSERT INTO
    character_contact_labels (character_id, label_id, name)
VALUES
    (?, ?, ?);

-- name: DeleteCharacterContactLabels :exec
DELETE FROM character_contact_labels
WHERE
    character_id = ?
    AND name IN (sqlc.slice('names'));

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
    name
FROM
    character_contact_labels
WHERE
    character_id = ?;
