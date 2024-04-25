-- name: CreateCharacter :one
INSERT INTO characters (
    id,
    alliance_id,
    birthday,
    corporation_id,
    description,
    faction_id,
    gender,
    last_login_at,
    name,
    race_id,
    security_status,
    ship_id,
    skill_points,
    location_id,
    wallet_balance
)
VALUES (
    ?, ?, ?, ?, ? ,?, ?, ?, ?, ?, ? ,?, ?, ?, ?
)
RETURNING *;

-- name: DeleteCharacter :exec
DELETE FROM characters
WHERE id = ?;

-- name: GetCharacter :one
SELECT
    sqlc.embed(characters),
    sqlc.embed(corporations),
    sqlc.embed(eve_races),
    sqlc.embed(eve_categories),
    sqlc.embed(eve_groups),
    sqlc.embed(eve_types),
    sqlc.embed(eve_regions),
    sqlc.embed(eve_constellations),
    sqlc.embed(eve_solar_systems),
    alliances.name as alliance_name,
    factions.name as faction_name
FROM characters
JOIN eve_entities AS corporations ON corporations.id = characters.corporation_id
JOIN eve_regions ON eve_regions.id = eve_constellations.eve_region_id
JOIN eve_constellations ON eve_constellations.id = eve_solar_systems.eve_constellation_id
JOIN eve_solar_systems ON eve_solar_systems.id = characters.location_id
JOIN eve_races ON eve_races.id = characters.race_id
JOIN eve_types ON eve_types.id = characters.ship_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
LEFT JOIN eve_entities AS alliances ON alliances.id = characters.alliance_id
LEFT JOIN eve_entities AS factions ON factions.id = characters.faction_id
WHERE characters.id = ?;

-- name: ListCharacters :many
SELECT
    characters.*,
    corporations.name as corporation_name,
    alliances.name as alliance_name,
    factions.name as faction_name
FROM characters
JOIN eve_entities AS corporations ON corporations.id = characters.corporation_id
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
    name = ?,
    security_status = ?,
    ship_id = ?,
    skill_points = ?,
    location_id = ?,
    wallet_balance = ?
WHERE id = ?;

-- name: UpdateOrCreateCharacter :one
INSERT INTO characters (
    id,
    alliance_id,
    birthday,
    corporation_id,
    description,
    faction_id,
    gender,
    last_login_at,
    name,
    race_id,
    security_status,
    ship_id,
    skill_points,
    location_id,
    wallet_balance
)
VALUES (
    ?1, ?2, ?3, ?4, ?5 ,?6, ?7, ?8, ?9, ?10, ?11 ,?12, ?13, ?14, ?15
)
ON CONFLICT(id) DO
UPDATE SET
    alliance_id = ?2,
    corporation_id = ?4,
    description = ?5,
    faction_id = ?6,
    last_login_at = ?8,
    name = ?9,
    security_status = ?11,
    ship_id = ?12,
    skill_points = ?13,
    location_id = ?14,
    wallet_balance = ?15
WHERE id = ?1
RETURNING *;
