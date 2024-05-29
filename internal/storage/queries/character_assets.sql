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
    et.name as eve_type_name
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
WHERE character_id = ?
AND item_id = ?;

-- name: ListCharacterAssetIDs :many
SELECT item_id
FROM character_assets
WHERE character_id = ?;

-- name: ListCharacterAssets :many
SELECT
    sqlc.embed(ca),
    et.name as eve_type_name
FROM character_assets ca
JOIN eve_types et ON et.id = ca.eve_type_id
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
