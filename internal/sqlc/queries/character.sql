-- name: CreateCharacter :one
INSERT INTO characters (
    alliance_id,
    corporation_id,
    description,
    faction_id,
    last_login_at,
    mail_updated_at,
    name,
    race_id,
    security_status,
    skill_points,
    solar_system_id,
    wallet_balance,
    id,
    birthday,
    gender
)
VALUES (
    ?, ?, ?, ?, ? ,?, ?, ?, ?, ?, ? ,?, ?, ?, ?
)
RETURNING *;

-- name: DeleteCharacter :exec
DELETE FROM characters
WHERE id = ?;

-- name: GetCharacter :one
SELECT characters.*, corporations.*, alliances.*, factions.*, races.Name as race_name, systems.*
FROM characters
JOIN eve_entities AS corporations ON corporations.id = characters.corporation_id
JOIN eve_entities AS systems ON systems.id = characters.solar_system_id
JOIN races ON races.id = characters.race_id
LEFT JOIN eve_entities AS alliances ON alliances.id = characters.alliance_id
LEFT JOIN eve_entities AS factions ON factions.id = characters.faction_id
WHERE characters.id = ?;

-- name: ListCharacters :many
SELECT characters.*, corporations.*, alliances.*, factions.*, races.Name as race_name, systems.*
FROM characters
JOIN eve_entities AS corporations ON corporations.id = characters.corporation_id
JOIN races ON races.id = characters.race_id
JOIN eve_entities AS systems ON systems.id = characters.solar_system_id
LEFT JOIN eve_entities AS alliances ON alliances.id = characters.alliance_id
LEFT JOIN eve_entities AS factions ON factions.id = characters.faction_id
ORDER BY characters.name;

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
    last_login_at = ?,
    mail_updated_at = ?,
    name = ?,
    security_status = ?,
    skill_points = ?,
    solar_system_id = ?,
    wallet_balance = ?
WHERE id = ?;
