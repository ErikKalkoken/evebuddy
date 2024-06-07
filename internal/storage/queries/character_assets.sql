-- name: CreateCharacterAsset :exec
INSERT INTO character_assets (
    character_id,
    eve_type_id,
    is_blueprint_copy,
    is_singleton,
    item_id,
    location_flag,
    location_id,
    location_type,
    name,
    quantity
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: DeleteExcludedCharacterAssets :exec
DELETE FROM character_assets
WHERE character_id = ?
AND item_id NOT IN (sqlc.slice('item_ids'));

-- name: GetCharacterAsset :one
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE character_id = ?
AND item_id = ?;

-- name: ListCharacterAssetIDs :many
SELECT item_id
FROM character_assets
WHERE character_id = ?;

-- name: ListAllCharacterAssets :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
ORDER BY et.name;

-- name: ListCharacterAssetsInLocation :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE character_id = ?
AND location_id = ?
ORDER BY et.id;

-- name: ListCharacterAssetsInShipHangar :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE character_id = ?
AND location_id = ?
AND location_flag = ?
AND eg.eve_category_id = ?
ORDER BY et.id;

-- name: ListCharacterAssetsInItemHangar :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE character_id = ?
AND location_id = ?
AND location_flag = ?
AND eg.eve_category_id != ?
ORDER BY et.id;

-- name: ListCharacterAssetLocations :many
SELECT DISTINCT
    ca.character_id,
    ca.location_type,
    ca.location_id,
    lo.name as location_name,
    sys.id as system_id,
    sys.name as system_name
FROM character_assets ca
JOIN eve_locations lo ON lo.id = ca.location_id
LEFT JOIN eve_solar_systems sys ON sys.id = lo.eve_solar_system_id
WHERE character_id = ?
AND location_flag = ?
ORDER BY location_name;

-- name: UpdateCharacterAsset :exec
UPDATE character_assets
SET
    location_flag = ?,
    location_id = ?,
    location_type = ?,
    name = ?,
    quantity = ?
WHERE character_id = ?
AND item_id = ?;
