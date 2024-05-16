-- name: CreateMyCharacter :one
INSERT INTO my_characters (
    id,
    last_login_at,
    ship_id,
    skill_points,
    location_id,
    wallet_balance
)
VALUES (
    ?, ?, ?, ?, ? ,?
)
RETURNING *;

-- name: DeleteMyCharacter :exec
DELETE FROM my_characters
WHERE id = ?;

-- name: GetMyCharacter :one
SELECT
    sqlc.embed(my_characters),
    sqlc.embed(eve_characters),
    sqlc.embed(corporations),
    sqlc.embed(eve_races),
    sqlc.embed(eve_character_alliances),
    sqlc.embed(eve_character_factions),
    location_id,
    ship_id
FROM my_characters
JOIN eve_characters ON eve_characters.id = my_characters.id
JOIN eve_entities AS corporations ON corporations.id = eve_characters.corporation_id
JOIN eve_races ON eve_races.id = eve_characters.race_id
LEFT JOIN eve_character_alliances ON eve_character_alliances.id = eve_characters.alliance_id
LEFT JOIN eve_character_factions ON eve_character_factions.id = eve_characters.faction_id
WHERE my_characters.id = ?;

-- name: ListMyCharacters :many
SELECT DISTINCT
    sqlc.embed(my_characters),
    sqlc.embed(eve_characters),
    sqlc.embed(corporations),
    sqlc.embed(eve_races),
    sqlc.embed(eve_character_alliances),
    sqlc.embed(eve_character_factions),
    location_id,
    ship_id
FROM my_characters
JOIN eve_characters ON eve_characters.id = my_characters.id
JOIN eve_entities AS corporations ON corporations.id = eve_characters.corporation_id
JOIN eve_races ON eve_races.id = eve_characters.race_id
LEFT JOIN eve_character_alliances ON eve_character_alliances.id = eve_characters.alliance_id
LEFT JOIN eve_character_factions ON eve_character_factions.id = eve_characters.faction_id
ORDER BY eve_characters.name;

-- name: ListMyCharactersShort :many
SELECT DISTINCT eve_characters.id, eve_characters.name, corporations.name
FROM my_characters
JOIN eve_characters ON eve_characters.id = my_characters.id
JOIN eve_entities AS corporations ON corporations.id = eve_characters.corporation_id
ORDER BY eve_characters.name;

-- name: ListMyCharacterIDs :many
SELECT id
FROM my_characters;

-- name: UpdateMyCharacter :exec
UPDATE my_characters
SET
    last_login_at = ?,
    ship_id = ?,
    skill_points = ?,
    location_id = ?,
    wallet_balance = ?
WHERE id = ?;

-- name: UpdateOrCreateMyCharacter :exec
INSERT INTO my_characters (
    id,
    last_login_at,
    ship_id,
    skill_points,
    location_id,
    wallet_balance
)
VALUES (
    ?1, ?2, ?3, ?4, ?5 ,?6
)
ON CONFLICT(id) DO
UPDATE SET
    last_login_at = ?2,
    ship_id = ?3,
    skill_points = ?4,
    location_id = ?5,
    wallet_balance = ?6
WHERE id = ?1;
