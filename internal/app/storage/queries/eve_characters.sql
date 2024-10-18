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
    sqlc.embed(ec),
    sqlc.embed(eec),
    sqlc.embed(er),
    eea.name as alliance_name,
    eea.category as alliance_category,
    eef.name as faction_name,
    eef.category as faction_category
FROM eve_characters ec
JOIN eve_entities AS eec ON eec.id = ec.corporation_id
JOIN eve_races er ON er.id = ec.race_id
LEFT JOIN eve_entities as eea ON eea.id = ec.alliance_id
LEFT JOIN eve_entities as eef ON eef.id = ec.faction_id
WHERE ec.id = ?;

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
