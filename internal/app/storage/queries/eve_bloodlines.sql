-- name: CreateEveBloodline :exec
INSERT INTO
    eve_bloodlines (
        id,
        charisma,
        corporation_id,
        description,
        intelligence,
        memory,
        name,
        perception,
        race_id,
        ship_type_id,
        willpower
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetEveBloodline :one
SELECT
    sqlc.embed(eb),
    sqlc.embed(eec),
    sqlc.embed(er),
    eea.name as faction_name
FROM
    eve_bloodlines eb
    JOIN eve_entities eec ON eec.id = eb.corporation_id
    JOIN eve_races er ON er.id = eb.race_id
    LEFT JOIN eve_entities eea ON eea.id = er.faction_id
WHERE
    eb.id = ?;

-- name: ListEveBloodlineIDs :many
SELECT
    id
FROM
    eve_bloodlines;
