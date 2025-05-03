-- name: CreateCharacterRole :exec
INSERT INTO character_roles (
    character_id,
    name
)
VALUES (
    ?, ?
);

-- name: DeleteCharacterRole :exec
DELETE FROM character_roles
WHERE character_id = ?
AND name = ?;

-- name: ListCharacterRoles :many
SELECT name
FROM character_roles
WHERE character_id = ?;
