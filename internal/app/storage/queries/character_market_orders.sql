-- name: GetCharacterMarketOrder :one
SELECT
    sqlc.embed(cmo),
    et.name AS type_name,
    el.name AS location_name
FROM
    character_market_orders cmo
    JOIN eve_types et ON et.id = cmo.type_id
    JOIN eve_locations el ON el.id = cmo.location_id
WHERE
    character_id = ?
    AND order_id = ?;

-- name: ListCharacterMarketOrders :many
SELECT
    sqlc.embed(cmo),
    et.name AS type_name,
    el.name AS location_name
FROM
    character_market_orders cmo
    JOIN eve_types et ON et.id = cmo.type_id
    JOIN eve_locations el ON el.id = cmo.location_id
WHERE
    character_id = ?;

-- name: ListAllCharacterMarketOrders :many
SELECT
    sqlc.embed(cmo),
    et.name AS type_name,
    el.name AS location_name
FROM
    character_market_orders cmo
    JOIN eve_types et ON et.id = cmo.type_id
    JOIN eve_locations el ON el.id = cmo.location_id;

-- name: UpdateOrCreateCharacterMarketOrder :exec
INSERT INTO
    character_market_orders (
        character_id,
        duration,
        escrow,
        is_buy_order,
        is_corporation,
        issued,
        location_id,
        min_volume,
        order_id,
        price,
        range,
        region_id,
        state,
        type_id,
        volume_remains,
        volume_total
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
        ?15,
        ?16
    )
ON CONFLICT (character_id, order_id) DO UPDATE
SET
    state = ?13,
    volume_remains = ?15;
