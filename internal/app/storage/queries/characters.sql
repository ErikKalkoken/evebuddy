-- name: DeleteCharacter :exec
DELETE FROM characters
WHERE id = ?;

-- name: GetCharacter :one
SELECT
    sqlc.embed(characters),
    sqlc.embed(eve_characters),
    sqlc.embed(corporations),
    sqlc.embed(eve_races),
    sqlc.embed(eve_character_alliances),
    sqlc.embed(eve_character_factions),
    home_id,
    location_id,
    ship_id
FROM characters
JOIN eve_characters ON eve_characters.id = characters.id
JOIN eve_entities AS corporations ON corporations.id = eve_characters.corporation_id
JOIN eve_races ON eve_races.id = eve_characters.race_id
LEFT JOIN eve_character_alliances ON eve_character_alliances.id = eve_characters.alliance_id
LEFT JOIN eve_character_factions ON eve_character_factions.id = eve_characters.faction_id
WHERE characters.id = ?;

-- name: ListCharacters :many
SELECT DISTINCT
    sqlc.embed(characters),
    sqlc.embed(eve_characters),
    sqlc.embed(corporations),
    sqlc.embed(eve_races),
    sqlc.embed(eve_character_alliances),
    sqlc.embed(eve_character_factions),
    home_id,
    location_id,
    ship_id
FROM characters
JOIN eve_characters ON eve_characters.id = characters.id
JOIN eve_entities AS corporations ON corporations.id = eve_characters.corporation_id
JOIN eve_races ON eve_races.id = eve_characters.race_id
LEFT JOIN eve_character_alliances ON eve_character_alliances.id = eve_characters.alliance_id
LEFT JOIN eve_character_factions ON eve_character_factions.id = eve_characters.faction_id
ORDER BY eve_characters.name;

-- name: ListCharactersShort :many
SELECT DISTINCT eve_characters.id, eve_characters.name
FROM characters
JOIN eve_characters ON eve_characters.id = characters.id
ORDER BY eve_characters.name;

-- name: ListCharacterIDs :many
SELECT id
FROM characters;

-- name: UpdateCharacterHomeId :exec
UPDATE characters
SET
    home_id = ?
WHERE id = ?;

-- name: UpdateCharacterLastLoginAt :exec
UPDATE characters
SET
    last_login_at = ?
WHERE id = ?;

-- name: UpdateCharacterLocationID :exec
UPDATE characters
SET
    location_id = ?
WHERE id = ?;

-- name: UpdateCharacterShipID :exec
UPDATE characters
SET
    ship_id = ?
WHERE id = ?;

-- name: UpdateCharacterSP :exec
UPDATE characters
SET
    total_sp = ?,
    unallocated_sp = ?
WHERE id = ?;

-- name: UpdateCharacterWalletBalance :exec
UPDATE characters
SET
    wallet_balance = ?
WHERE id = ?;

-- name: UpdateOrCreateCharacter :exec
INSERT INTO characters (
    id,
    home_id,
    last_login_at,
    location_id,
    ship_id,
    total_sp,
    unallocated_sp,
    wallet_balance
)
VALUES (
    ?1, ?2, ?3, ?4, ?5 ,?6, ?7, ?8
)
ON CONFLICT(id) DO
UPDATE SET
    home_id = ?2,
    last_login_at = ?3,
    location_id = ?4,
    ship_id = ?5,
    total_sp = ?6,
    unallocated_sp = ?7,
    wallet_balance = ?8
WHERE id = ?1;
