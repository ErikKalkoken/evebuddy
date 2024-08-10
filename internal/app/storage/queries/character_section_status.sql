-- name: GetCharacterSectionStatus :one
SELECT *
FROM character_section_status
WHERE character_id = ?
AND section_id = ?;

-- name: ListCharacterSectionStatus :many
SELECT *
FROM character_section_status
WHERE character_id = ?;

-- name: UpdateOrCreateCharacterSectionStatus :one
INSERT INTO character_section_status (
    character_id,
    section_id,
    completed_at,
    content_hash,
    error,
    started_at,
    updated_at
)
VALUES (
    ?1, ?2, ?3, ?4, ?5, ?6, ?7
)
ON CONFLICT(character_id, section_id) DO
UPDATE SET
    completed_at = ?3,
    content_hash = ?4,
    error = ?5,
    started_at = ?6,
    updated_at = ?7
WHERE character_id = ?1
AND section_id = ?2
RETURNING *;
