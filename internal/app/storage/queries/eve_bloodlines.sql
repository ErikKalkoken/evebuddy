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
    sqlc.embed(ee),
    sqlc.embed(er)
FROM
    eve_bloodlines eb
    JOIN eve_entities ee ON ee.id = eb.corporation_id
    JOIN eve_races er ON er.id = eb.race_id
WHERE
    eb.id = ?;

-- name: ListEveBloodlineIDs :many
SELECT
    id
FROM
    eve_bloodlines;
