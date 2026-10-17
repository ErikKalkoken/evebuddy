-- name: GetCharacterSectionStatus :one
SELECT
    *
FROM
    character_section_status
WHERE
    character_id = ?
    AND section_id = ?;

-- name: CountCharactersWithSection :one
SELECT
    COUNT(*)
FROM
    character_section_status
WHERE
    section_id = ?
    AND completed_at IS NOT NULL;

-- name: ListCharacterSectionStatus :many
SELECT
    *
FROM
    character_section_status
WHERE
    character_id = ?;

-- name: UpdateOrCreateCharacterSectionStatus :one
INSERT INTO
    character_section_status (
        character_id,
        section_id,
        completed_at,
        content_hash,
        error,
        started_at,
        updated_at
    )
VALUES
    (?1, ?2, ?3, ?4, ?5, ?6, ?7)
ON CONFLICT (character_id, section_id) DO UPDATE
SET
    completed_at = ?3,
    content_hash = ?4,
    error = ?5,
    started_at = ?6,
    updated_at = ?7
RETURNING
    *;

-- name: ClearCharacterSectionStatusStartedAt :many
UPDATE character_section_status
SET
    started_at = NULL
WHERE
    started_at IS NOT NULL
RETURNING
    *;
