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
    sqlc.embed(ec),
    average_price as price
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
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
    sqlc.embed(ec),
    average_price as price
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE;

-- name: ListCharacterAssets :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
WHERE character_id = ?;

-- name: ListCharacterAssetsInLocation :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
WHERE character_id = ?
AND location_id = ?
ORDER BY et.id, ca.name;

-- name: ListCharacterAssetsInShipHangar :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
WHERE character_id = ?
AND location_id = ?
AND location_flag = ?
AND eg.eve_category_id = ?
ORDER BY et.id, ca.name;

-- name: ListCharacterAssetsInItemHangar :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
WHERE character_id = ?
AND location_id = ?
AND location_flag = ?
AND eg.eve_category_id != ?
ORDER BY et.id, ca.name;

-- name: CalculateCharacterAssetTotalValue :one
SELECT SUM(IFNULL(emp.average_price, 0) * quantity * IIF(ca.is_blueprint_copy IS TRUE, 0, 1)) as total
FROM character_assets ca
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id
WHERE character_id = ?;

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
