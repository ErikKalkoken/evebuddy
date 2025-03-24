-- name: CreateCharacterWalletTransaction :exec
INSERT INTO
    character_wallet_transactions (
        client_id,
        date,
        eve_type_id,
        is_buy,
        is_personal,
        journal_ref_id,
        character_id,
        location_id,
        quantity,
        transaction_id,
        unit_price
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetCharacterWalletTransaction :one
SELECT
    sqlc.embed(cwt),
    sqlc.embed(ee),
    et.name as eve_type_name,
    el.name as location_name,
    ess.security_status as system_security_status
FROM
    character_wallet_transactions cwt
    JOIN eve_entities ee ON ee.id = cwt.client_id
    JOIN eve_types et ON et.id = cwt.eve_type_id
    JOIN eve_locations el ON el.id = cwt.location_id
    LEFT JOIN eve_solar_systems ess ON ess.id = el.eve_solar_system_id
WHERE
    character_id = ?
    and transaction_id = ?;

-- name: ListCharacterWalletTransactions :many
SELECT
    sqlc.embed(cwt),
    sqlc.embed(ee),
    et.name as eve_type_name,
    el.name as location_name,
    ess.security_status as system_security_status
FROM
    character_wallet_transactions cwt
    JOIN eve_entities ee ON ee.id = cwt.client_id
    JOIN eve_types et ON et.id = cwt.eve_type_id
    JOIN eve_locations el ON el.id = cwt.location_id
    LEFT JOIN eve_solar_systems ess ON ess.id = el.eve_solar_system_id
WHERE
    character_id = ?
ORDER BY
    date DESC;

-- name: ListCharacterWalletTransactionIDs :many
SELECT
    transaction_id
FROM
    character_wallet_transactions
WHERE
    character_id = ?;