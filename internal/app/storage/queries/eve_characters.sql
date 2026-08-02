-- name: UpdateOrCreateEveCharacter :exec
INSERT INTO
    eve_characters (
        id,
        alliance_id,
        birthday,
        bloodline_id,
        corporation_id,
        description,
        faction_id,
        gender,
        name,
        race_id,
        security_status,
        title
    )
VALUES
    (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)
ON CONFLICT (id) DO UPDATE
SET
    alliance_id = ?2,
    birthday = ?3,
    bloodline_id = ?4,
    corporation_id = ?5,
    description = ?6,
    faction_id = ?7,
    gender = ?8,
    name = ?9,
    race_id = ?10,
    security_status = ?11,
    title = ?12;

-- name: DeleteEveCharacter :exec
DELETE FROM eve_characters
WHERE
    id = ?;

-- name: GetEveCharacter :one
SELECT
    sqlc.embed(ec),
    sqlc.embed(eec),
    sqlc.embed(er),
    eea.name as alliance_name,
    eea.category as alliance_category,
    eef.name as faction_name,
    eef.category as faction_category,
    eb.id as bloodline_id,
    eb.name as bloodline_name
FROM
    eve_characters ec
    JOIN eve_entities eec ON eec.id = ec.corporation_id
    JOIN eve_races er ON er.id = ec.race_id
    LEFT JOIN eve_entities eea ON eea.id = ec.alliance_id
    LEFT JOIN eve_entities eef ON eef.id = ec.faction_id
    LEFT JOIN eve_bloodlines eb ON eb.id = ec.bloodline_id
WHERE
    ec.id = ?;

-- name: ListEveCharacterIDs :many
SELECT
    id
FROM
    eve_characters;

-- name: UpdateEveCharacterName :exec
UPDATE eve_characters
SET
    name = ?
WHERE
    id = ?;