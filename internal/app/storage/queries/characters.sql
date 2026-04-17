-- name: CreateCharacter :exec
INSERT INTO
    characters (
        id,
        asset_value,
        contracts_escrow,
        contract_items_value,
        home_id,
        is_training_watched,
        last_clone_jump_at,
        last_login_at,
        location_id,
        orders_escrow,
        order_items_value,
        ship_id,
        total_sp,
        unallocated_sp,
        wallet_balance
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteCharacter :exec
DELETE FROM characters
WHERE
    id = ?;

-- name: DisableAllTrainingWatchers :exec
UPDATE characters
SET
    is_training_watched = FALSE;

-- name: GetCharacter :one
SELECT
    sqlc.embed(cc),
    sqlc.embed(ec),
    sqlc.embed(eec),
    sqlc.embed(er),
    eea.name as alliance_name,
    eea.category as alliance_category,
    eef.name as faction_name,
    eef.category as faction_category,
    home_id,
    location_id,
    ship_id
FROM
    characters cc
    JOIN eve_characters ec ON ec.id = cc.id
    JOIN eve_entities eec ON eec.id = ec.corporation_id
    JOIN eve_races er ON er.id = ec.race_id
    LEFT JOIN eve_entities eea ON eea.id = ec.alliance_id
    LEFT JOIN eve_entities eef ON eef.id = ec.faction_id
WHERE
    cc.id = ?;

-- name: GetCharacterAssetValue :one
SELECT
    asset_value
FROM
    characters
WHERE
    id = ?;

-- name: ListCharacters :many
SELECT DISTINCT
    sqlc.embed(cc),
    sqlc.embed(ec),
    sqlc.embed(eec),
    sqlc.embed(er),
    eea.name as alliance_name,
    eea.category as alliance_category,
    eef.name as faction_name,
    eef.category as faction_category,
    home_id,
    location_id,
    ship_id
FROM
    characters cc
    JOIN eve_characters ec ON ec.id = cc.id
    JOIN eve_entities eec ON eec.id = ec.corporation_id
    JOIN eve_races er ON er.id = ec.race_id
    LEFT JOIN eve_entities eea ON eea.id = ec.alliance_id
    LEFT JOIN eve_entities eef ON eef.id = ec.faction_id
ORDER BY
    ec.name;

-- name: ListCharactersShort :many
SELECT DISTINCT
    ec.id,
    ec.name
FROM
    characters c
    JOIN eve_characters ec ON ec.id = c.id
ORDER BY
    ec.name;

-- name: ListCharacterIDs :many
SELECT
    id
FROM
    characters;

-- name: ListCharacterCorporations :many
SELECT DISTINCT
    ee.id,
    ee.name
FROM
    characters ch
    JOIN eve_characters ec ON ec.id = ch.id
    JOIN corporations ep ON ep.id = ec.corporation_id
    JOIN eve_entities ee ON ee.id = ec.corporation_id;

-- name: ListCharacterCorporationIDs :many
SELECT DISTINCT
    ec.corporation_id
FROM
    characters ch
    JOIN eve_characters ec ON ec.id = ch.id
    JOIN eve_entities ee ON ee.id = ec.corporation_id;

-- name: ListCharacterWealthValues :many
SELECT
    id,
    asset_value,
    wallet_balance
FROM
    characters;

-- name: UpdateCharacterAssetValue :exec
UPDATE characters
SET
    asset_value = ?
WHERE
    id = ?;

-- name: UpdateCharacterContractsEscrow :exec
UPDATE characters
SET
    contracts_escrow = ?
WHERE
    id = ?;

-- name: UpdateCharacterContractItemsValue :exec
UPDATE characters
SET
    contract_items_value = ?
WHERE
    id = ?;

-- name: UpdateCharacterLastCloneJump :exec
UPDATE characters
SET
    last_clone_jump_at = ?
WHERE
    id = ?;

-- name: UpdateCharacterHomeId :exec
UPDATE characters
SET
    home_id = ?
WHERE
    id = ?;

-- name: UpdateCharacterIsTrainingWatched :exec
UPDATE characters
SET
    is_training_watched = ?
WHERE
    id = ?;

-- name: UpdateCharacterLastLoginAt :exec
UPDATE characters
SET
    last_login_at = ?
WHERE
    id = ?;

-- name: UpdateCharacterLocationID :exec
UPDATE characters
SET
    location_id = ?
WHERE
    id = ?;

-- name: UpdateCharacterOrdersEscrow :exec
UPDATE characters
SET
    orders_escrow = ?
WHERE
    id = ?;

-- name: UpdateCharacterOrderItemsValue :exec
UPDATE characters
SET
    order_items_value = ?
WHERE
    id = ?;

-- name: UpdateCharacterShipID :exec
UPDATE characters
SET
    ship_id = ?
WHERE
    id = ?;

-- name: UpdateCharacterSP :exec
UPDATE characters
SET
    total_sp = ?,
    unallocated_sp = ?
WHERE
    id = ?;

-- name: UpdateCharacterWalletBalance :exec
UPDATE characters
SET
    wallet_balance = ?
WHERE
    id = ?;
