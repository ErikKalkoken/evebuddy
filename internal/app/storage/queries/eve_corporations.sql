-- name: GetEveCorporation :one
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
    eve_corporations ec
    LEFT JOIN eve_entities AS eec ON eec.id = ec.ceo_id
    LEFT JOIN eve_entities AS eer ON eer.id = ec.creator_id
    LEFT JOIN eve_entities as eea ON eea.id = ec.alliance_id
    LEFT JOIN eve_entities as eef ON eef.id = ec.faction_id
    LEFT JOIN eve_entities as eeh ON eeh.id = ec.home_station_id
WHERE
    ec.id = ?;

-- name: ListEveCorporationIDs :many
SELECT
    id
FROM
    eve_corporations;

-- name: UpdateOrCreateEveCorporation :exec
INSERT INTO
    eve_corporations (
        id,
        alliance_id,
        ceo_id,
        creator_id,
        date_founded,
        description,
        faction_id,
        home_station_id,
        member_count,
        name,
        shares,
        tax_rate,
        ticker,
        url,
        war_eligible
    )
VALUES
    (
        ?1,
        ?2,
        ?3,
        ?4,
        ?5,
        ?6,
        ?7,
        ?8,
        ?9,
        ?10,
        ?11,
        ?12,
        ?13,
        ?14,
        ?15
    )
ON CONFLICT (id) DO UPDATE
SET
    alliance_id = ?2,
    ceo_id = ?3,
    description = ?6,
    faction_id = ?7,
    home_station_id = ?8,
    member_count = ?9,
    name = ?10,
    shares = ?11,
    tax_rate = ?12,
    ticker = ?13,
    url = ?14,
    war_eligible = ?15;

-- name: UpdateEveCorporationName :exec
UPDATE eve_corporations
SET
    name = ?
WHERE
    id = ?;