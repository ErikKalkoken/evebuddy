-- name: CalculateCharacterOrderItemsValue :one
SELECT
    SUM(IFNULL(mp.average_price, 0) * volume_remains)
FROM
    character_market_orders cmo
    JOIN eve_types et ON et.id = cmo.type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    LEFT JOIN eve_market_prices mp ON mp.type_id = cmo.type_id
WHERE
    character_id = ?
    AND is_buy_order IS FALSE
    AND state IN (sqlc.slice('states'))
    AND eg.eve_category_id <> ?;

-- name: CalculateCharacterOrdersEscrow :one
SELECT
    SUM(escrow)
FROM
    character_market_orders cmo
WHERE
    character_id = ?
    AND is_buy_order IS TRUE
    AND state IN (sqlc.slice('states'));

-- name: DeleteCharacterMarketOrders :exec
DELETE FROM character_market_orders
WHERE
    character_id = ?
    AND order_id IN (sqlc.slice('order_ids'));

-- name: GetCharacterMarketOrder :one
SELECT
    sqlc.embed(cmo),
    sqlc.embed(ee),
    et.name AS type_name,
    el.name AS location_name,
    er.name AS region_name,
    els.security_status as location_security
FROM
    character_market_orders cmo
    JOIN eve_entities ee ON ee.id = cmo.owner_id
    JOIN eve_locations el ON el.id = cmo.location_id
    JOIN eve_regions er ON er.id = cmo.region_id
    JOIN eve_types et ON et.id = cmo.type_id
    LEFT JOIN eve_solar_systems els ON els.id = el.eve_solar_system_id
WHERE
    character_id = ?
    AND order_id = ?;

-- name: ListCharacterMarketOrderIDs :many
SELECT
    order_id
FROM
    character_market_orders
WHERE
    character_id = ?;

-- name: ListCharacterMarketOrders :many
SELECT
    sqlc.embed(cmo),
    sqlc.embed(ee),
    et.name AS type_name,
    el.name AS location_name,
    er.name AS region_name,
    els.security_status as location_security
FROM
    character_market_orders cmo
    JOIN eve_entities ee ON ee.id = cmo.owner_id
    JOIN eve_locations el ON el.id = cmo.location_id
    JOIN eve_regions er ON er.id = cmo.region_id
    JOIN eve_types et ON et.id = cmo.type_id
    LEFT JOIN eve_solar_systems els ON els.id = el.eve_solar_system_id
WHERE
    character_id = ?;

-- name: ListAllCharacterMarketOrders :many
SELECT
    sqlc.embed(cmo),
    sqlc.embed(ee),
    et.name AS type_name,
    el.name AS location_name,
    er.name AS region_name,
    els.security_status as location_security
FROM
    character_market_orders cmo
    JOIN eve_entities ee ON ee.id = cmo.owner_id
    JOIN eve_locations el ON el.id = cmo.location_id
    JOIN eve_regions er ON er.id = cmo.region_id
    JOIN eve_types et ON et.id = cmo.type_id
    LEFT JOIN eve_solar_systems els ON els.id = el.eve_solar_system_id
WHERE
    cmo.is_buy_order = ?;

-- name: UpdateCharacterMarketOrderState :exec
UPDATE character_market_orders
SET
    state = ?
WHERE
    character_id = ?
    AND order_id IN (sqlc.slice('order_ids'));

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
        owner_id,
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
        ?16,
        ?17
    )
ON CONFLICT (character_id, order_id) DO UPDATE
SET
    escrow = ?3,
    price = ?11,
    state = ?14,
    volume_remains = ?16;
