-- name: CreateCharacter :exec
INSERT INTO
    characters (
        id,
        home_id,
        last_login_at,
        location_id,
        ship_id,
        total_sp,
        unallocated_sp,
        wallet_balance,
        asset_value,
        is_training_watched,
        last_clone_jump_at
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteCharacter :exec
DELETE FROM
    characters
WHERE
    id = ?;

-- name: DisableAllTrainingWatchers :exec
UPDATE
    characters
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
SELECT
    DISTINCT sqlc.embed(cc),
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
SELECT
    DISTINCT eve_characters.id,
    eve_characters.name
FROM
    characters
    JOIN eve_characters ON eve_characters.id = characters.id
ORDER BY
    eve_characters.name;

-- name: ListCharacterIDs :many
SELECT
    id
FROM
    characters;

-- name: UpdateCharacterLastCloneJump :exec
UPDATE
    characters
SET
    last_clone_jump_at = ?
WHERE
    id = ?;

-- name: UpdateCharacterHomeId :exec
UPDATE
    characters
SET
    home_id = ?
WHERE
    id = ?;

-- name: UpdateCharacterIsTrainingWatched :exec
UPDATE
    characters
SET
    is_training_watched = ?
WHERE
    id = ?;

-- name: UpdateCharacterLastLoginAt :exec
UPDATE
    characters
SET
    last_login_at = ?
WHERE
    id = ?;

-- name: UpdateCharacterLocationID :exec
UPDATE
    characters
SET
    location_id = ?
WHERE
    id = ?;

-- name: UpdateCharacterShipID :exec
UPDATE
    characters
SET
    ship_id = ?
WHERE
    id = ?;

-- name: UpdateCharacterSP :exec
UPDATE
    characters
SET
    total_sp = ?,
    unallocated_sp = ?
WHERE
    id = ?;

-- name: UpdateCharacterWalletBalance :exec
UPDATE
    characters
SET
    wallet_balance = ?
WHERE
    id = ?;

-- name: UpdateCharacterAssetValue :exec
UPDATE
    characters
SET
    asset_value = ?
WHERE
    id = ?;
