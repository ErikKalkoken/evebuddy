-- name: DeleteCharacterLoyaltyPointEntries :exec
DELETE FROM character_loyalty_point_entries
WHERE
    character_id = ?
    AND corporation_id IN (sqlc.slice('corporation_ids'));

-- name: GetCharacterLoyaltyPointEntry :one
SELECT
    sqlc.embed(clp),
    ec.name as corporation_name,
    ec.faction_id as faction_id,
    eef.name as faction_name,
    eef.category as faction_category
FROM
    character_loyalty_point_entries clp
    JOIN eve_corporations ec ON ec.id = clp.corporation_id
    LEFT JOIN eve_entities as eef ON eef.id = ec.faction_id
WHERE
    character_id = ?
    AND corporation_id = ?;

-- name: ListCharacterLoyaltyPointEntryIDs :many
SELECT
    corporation_id
FROM
    character_loyalty_point_entries
WHERE
    character_id = ?;

-- name: ListCharacterLoyaltyPointEntries :many
SELECT
    sqlc.embed(clp),
    ec.name as corporation_name,
    ec.faction_id as faction_id,
    eef.name as faction_name,
    eef.category as faction_category
FROM
    character_loyalty_point_entries clp
    JOIN eve_corporations ec ON ec.id = clp.corporation_id
    LEFT JOIN eve_entities as eef ON eef.id = ec.faction_id
WHERE
    character_id = ?;

-- name: ListAllCharacterLoyaltyPointEntries :many
SELECT
    sqlc.embed(clp),
    ec.name as corporation_name,
    ec.faction_id as faction_id,
    eef.name as faction_name,
    eef.category as faction_category
FROM
    character_loyalty_point_entries clp
    JOIN eve_corporations ec ON ec.id = clp.corporation_id
    LEFT JOIN eve_entities as eef ON eef.id = ec.faction_id;

-- name: UpdateOrCreateCharacterLoyaltyPointEntry :exec
INSERT INTO
    character_loyalty_point_entries (character_id, corporation_id, loyalty_points)
VALUES
    (?1, ?2, ?3)
ON CONFLICT (character_id, corporation_id) DO UPDATE
SET
    loyalty_points = ?3;
