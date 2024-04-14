-- name: DeleteCharacter :exec
DELETE FROM characters
WHERE id = ?;

-- name: GetCharacter :one
SELECT sqlc.embed(characters), sqlc.embed(corporations), sqlc.embed(alliances), sqlc.embed(factions)
FROM characters
JOIN eve_entities AS corporations ON corporations.id = characters.corporation_id
LEFT JOIN eve_entities AS alliances ON alliances.id = characters.alliance_id
LEFT JOIN eve_entities AS factions ON factions.id = characters.faction_id
WHERE characters.id = ?;

-- name: GetFirstCharacter :one
SELECT sqlc.embed(characters), sqlc.embed(corporations), sqlc.embed(alliances), sqlc.embed(factions)
FROM characters
JOIN eve_entities AS corporations ON corporations.id = characters.corporation_id
LEFT JOIN eve_entities AS alliances ON alliances.id = characters.alliance_id
LEFT JOIN eve_entities AS factions ON factions.id = characters.faction_id
LIMIT 1;

-- name: ListCharacters :many
SELECT sqlc.embed(characters), sqlc.embed(corporations), sqlc.embed(alliances), sqlc.embed(factions)
FROM characters
JOIN eve_entities AS corporations ON corporations.id = characters.corporation_id
LEFT JOIN eve_entities AS alliances ON alliances.id = characters.alliance_id
LEFT JOIN eve_entities AS factions ON factions.id = characters.faction_id
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
