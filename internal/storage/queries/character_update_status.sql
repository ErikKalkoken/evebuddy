-- name: GetCharacterUpdateStatus :one
SELECT *
FROM character_update_status
WHERE character_id = ?
AND section_id = ?;

-- name: ListCharacterUpdateStatus :many
SELECT *
FROM character_update_status
WHERE character_id = ?;

-- name: SetCharacterUpdateStatus :exec
INSERT INTO character_update_status (
    character_id,
    section_id,
    content_hash,
    error
)
VALUES (
    ?1, ?2, "", ?3
)
ON CONFLICT(character_id, section_id) DO
UPDATE SET
    error = ?3
WHERE character_id = ?1
AND section_id = ?2;

-- name: UpdateOrCreateCharacterUpdateStatus :exec
INSERT INTO character_update_status (
    character_id,
    section_id,
    content_hash,
    error,
    last_updated_at
)
VALUES (
    ?1, ?2, ?3, ?4, ?5
)
ON CONFLICT(character_id, section_id) DO
UPDATE SET
    content_hash = ?3,
    error = ?4,
    last_updated_at = ?5
WHERE character_id = ?1
AND section_id = ?2;
