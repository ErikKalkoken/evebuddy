-- name: GetCharacterUpdateStatus :one
SELECT *
FROM character_update_status
WHERE character_id = ?
AND section_id = ?;

-- name: UpdateOrCreateCharacterUpdateStatus :exec
INSERT INTO character_update_status (
    character_id,
    section_id,
    updated_at,
    content_hash
)
VALUES (
    ?1, ?2, ?3, ?4
)
ON CONFLICT(character_id, section_id) DO
UPDATE SET
    updated_at = ?3,
    content_hash = ?4
WHERE character_id = ?1
AND section_id = ?2;
