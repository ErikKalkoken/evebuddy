-- name: DeleteCharacter :exec
DELETE FROM characters
WHERE id = ?;

-- name: GetCharacter :one
SELECT *
FROM characters
WHERE id = ?;

-- name: GetFirstCharacter :one
SELECT *
FROM characters
LIMIT 1;

-- name: ListCharacters :many
SELECT *
FROM characters
ORDER BY name;

-- name: ListCharacterIDs :many
SELECT id
FROM characters;

-- name: UpdateCharacter :exec
UPDATE characters
SET
    alliance_id = ?,
    corporation_id = ?,
    description = ?,
    faction_id = ?,
    mail_updated_at = ?,
    name = ?,
    security_status = ?,
    skill_points = ?,
    wallet_balance = ?;

-- name: UpdateOrCreateCharacter :one
INSERT INTO characters (
    alliance_id,
    corporation_id,
    description,
    faction_id,
    mail_updated_at,
    name,
    security_status,
    skill_points,
    wallet_balance,
    id,
    birthday,
    gender
)
VALUES (
    ?, ?, ?, ?, ? ,?, ?, ?, ?, ?, ? ,?
)
ON CONFLICT (id) DO
UPDATE SET
    alliance_id = ?,
    corporation_id = ?,
    description = ?,
    faction_id = ?,
    mail_updated_at = ?,
    name = ?,
    security_status = ?,
    skill_points = ?,
    wallet_balance = ?
RETURNING *;
