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

-- name: UpdateOrCreateCharacterContact :one
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
    standing = ?5
RETURNING id;

-- name: ListCharacterContactContactLabelIds :many
SELECT
    ccl.label_id
FROM
    character_contacts_labels map
    JOIN character_contacts cc ON cc.id = map.contact_id
    JOIN character_contact_labels ccl ON ccl.id = map.label_id
WHERE
    cc.character_id = ? AND cc.contact_id = ?;

-- name: ListCharacterContactContactLabels :many
SELECT
    ccl.name
FROM
    character_contacts_labels map
    JOIN character_contacts cc ON cc.id = map.contact_id
    JOIN character_contact_labels ccl ON ccl.id = map.label_id
WHERE
    cc.id = ?;

-- name: CreateCharacterContactContactLabel :exec
INSERT INTO
    character_contacts_labels (contact_id, label_id)
VALUES
    (?, ?);

-- name: DeleteCharacterContactContactLabels :exec
DELETE FROM character_contacts_labels
WHERE id IN (
    SELECT map.id
    FROM character_contacts_labels map
    JOIN character_contacts cc ON cc.id = map.contact_id
    JOIN character_contact_labels ccl ON ccl.id = map.label_id
    WHERE cc.character_id = ? AND cc.contact_id = ?
    AND ccl.label_id IN (sqlc.slice('label_ids'))
);
