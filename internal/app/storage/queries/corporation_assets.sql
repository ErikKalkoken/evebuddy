-- name: CreateCorporationAsset :exec
INSERT INTO corporation_assets (
    corporation_id,
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

-- name: DeleteCorporationAssets :exec
DELETE FROM corporation_assets
WHERE corporation_id = ?
AND item_id IN (sqlc.slice('item_ids'));

-- name: GetCorporationAsset :one
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM corporation_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
WHERE corporation_id = ?
AND item_id = ?;

-- name: ListCorporationAssetIDs :many
SELECT item_id
FROM corporation_assets
WHERE corporation_id = ?;

-- name: ListAllCorporationAssets :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM corporation_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE;

-- name: ListCorporationAssets :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM corporation_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
WHERE corporation_id = ?;

-- name: ListCorporationAssetsInLocation :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM corporation_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
WHERE corporation_id = ?
AND location_id = ?
ORDER BY et.id, ca.name;

-- name: ListCorporationAssetsInShipHangar :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM corporation_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
WHERE corporation_id = ?
AND location_id = ?
AND location_flag = ?
AND eg.eve_category_id = ?
ORDER BY et.id, ca.name;

-- name: ListCorporationAssetsInItemHangar :many
SELECT
    sqlc.embed(ca),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec),
    average_price as price
FROM corporation_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id AND ca.is_blueprint_copy IS FALSE
WHERE corporation_id = ?
AND location_id = ?
AND location_flag = ?
AND eg.eve_category_id != ?
ORDER BY et.id, ca.name;

-- name: CalculateCorporationAssetTotalValue :one
SELECT SUM(IFNULL(emp.average_price, 0) * quantity * IIF(ca.is_blueprint_copy IS TRUE, 0, 1)) as total
FROM corporation_assets ca
LEFT JOIN eve_market_prices emp ON emp.type_id = ca.eve_type_id
WHERE corporation_id = ?;

-- name: UpdateCorporationAsset :exec
UPDATE corporation_assets
SET
    location_flag = ?,
    location_id = ?,
    location_type = ?,
    quantity = ?
WHERE corporation_id = ?
AND item_id = ?;

-- name: UpdateCorporationAssetName :exec
UPDATE corporation_assets
SET
    name = ?
WHERE corporation_id = ?
AND item_id = ?;
