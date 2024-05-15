-- name: GetMyCharacterUpdateStatus :one
SELECT *
FROM my_character_update_status
WHERE my_character_id = ?
AND section_id = ?;

-- name: UpdateOrCreateMyCharacterUpdateStatus :exec
INSERT INTO my_character_update_status (
    my_character_id,
    section_id,
    updated_at,
    content_hash
)
VALUES (
    ?1, ?2, ?3, ?4
)
ON CONFLICT(my_character_id, section_id) DO
UPDATE SET
    updated_at = ?3,
    content_hash = ?4
WHERE my_character_id = ?1
AND section_id = ?2;
