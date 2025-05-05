-- name: CreateCorporation :exec
INSERT INTO
    corporations (id)
VALUES
    (?);

-- name: DeleteCorporation :exec
DELETE FROM corporations
WHERE
    id = ?;

-- name: ListOrphanedCorporationIDs :many
SELECT
    id
FROM
    corporations
WHERE
    id NOT IN (
        SELECT
            ec.corporation_id
        FROM
            characters ch
            JOIN eve_characters ec ON ec.id = ch.id
    );

-- name: GetCorporation :one
SELECT
    sqlc.embed(ec),
    eec.name as ceo_name,
    eec.category as ceo_category,
    eer.name as creator_name,
    eer.category as creator_category,
    eea.name as alliance_name,
    eea.category as alliance_category,
    eef.name as faction_name,
    eef.category as faction_category,
    eeh.name as station_name,
    eeh.category as station_category
FROM
    corporations co
    JOIN eve_corporations ec ON ec.id = co.id
    LEFT JOIN eve_entities AS eec ON eec.id = ec.ceo_id
    LEFT JOIN eve_entities AS eer ON eer.id = ec.creator_id
    LEFT JOIN eve_entities as eea ON eea.id = ec.alliance_id
    LEFT JOIN eve_entities as eef ON eef.id = ec.faction_id
    LEFT JOIN eve_entities as eeh ON eeh.id = ec.home_station_id
WHERE
    co.id = ?;

-- name: ListCorporationIDs :many
SELECT
    id
FROM
    corporations;

-- name: ListCorporationsShort :many
SELECT
    co.id,
    ec.name
FROM
    corporations co
    JOIN eve_corporations ec ON ec.id = co.id
ORDER BY
    ec.name;
