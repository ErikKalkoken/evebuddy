-- name: CreateCharacterContractItem :exec
INSERT INTO
    character_contract_items (
        contract_id,
        is_included,
        is_singleton,
        quantity,
        raw_quantity,
        record_id,
        type_id
    )
VALUES
    (
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?
    );

-- name: GetCharacterContractItem :one
SELECT
    sqlc.embed(cci),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM
    character_contract_items cci
    JOIN eve_types et ON et.id = cci.type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE
    contract_id = ?
    AND record_id = ?;

-- name: ListCharacterContractItems :many
SELECT
    sqlc.embed(cci),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM
    character_contract_items cci
    JOIN eve_types et ON et.id = cci.type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE
    contract_id = ?;