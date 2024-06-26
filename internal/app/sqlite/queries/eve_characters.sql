-- name: CreateEveCharacter :exec
INSERT INTO eve_characters (
    id,
    alliance_id,
    birthday,
    corporation_id,
    description,
    faction_id,
    gender,
    name,
    race_id,
    security_status,
    title
)
VALUES (
    ?, ?, ?, ?, ? ,?, ?, ?, ?, ?, ?
);

-- name: DeleteEveCharacter :exec
DELETE FROM eve_characters
WHERE id = ?;

-- name: GetEveCharacter :one
SELECT
    sqlc.embed(eve_characters),
    sqlc.embed(corporations),
    sqlc.embed(eve_races),
    sqlc.embed(eve_character_alliances),
    sqlc.embed(eve_character_factions)
FROM eve_characters
JOIN eve_entities AS corporations ON corporations.id = eve_characters.corporation_id
JOIN eve_races ON eve_races.id = eve_characters.race_id
LEFT JOIN eve_character_alliances ON eve_character_alliances.id = eve_characters.alliance_id
LEFT JOIN eve_character_factions ON eve_character_factions.id = eve_characters.faction_id
WHERE eve_characters.id = ?;

-- name: ListEveCharacterIDs :many
SELECT id
FROM eve_characters;

-- name: UpdateEveCharacter :exec
UPDATE eve_characters
SET
    alliance_id = ?,
    corporation_id = ?,
    description = ?,
    faction_id = ?,
    name = ?,
    security_status = ?,
    title = ?
WHERE id = ?;
