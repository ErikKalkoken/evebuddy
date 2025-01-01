-- name: CreateCharacterContract :one
INSERT INTO
    character_contracts (
        acceptor_id,
        assignee_id,
        availability,
        buyout,
        character_id,
        collateral,
        contract_id,
        date_accepted,
        date_completed,
        date_expired,
        date_issued,
        days_to_complete,
        end_location_id,
        for_corporation,
        issuer_corporation_id,
        issuer_id,
        price,
        reward,
        start_location_id,
        status,
        title,
        type,
        volume
    )
VALUES
    (
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?
    ) RETURNING id;

-- name: GetCharacterContract :one
SELECT
    sqlc.embed(cc),
    sqlc.embed(issuer_corporation),
    sqlc.embed(issuer),
    acceptor.name as acceptor_name,
    acceptor.category as acceptor_category,
    assignee.name as assignee_name,
    assignee.category as assignee_category
FROM
    character_contracts cc
    JOIN eve_entities AS issuer_corporation ON issuer_corporation.id = cc.issuer_corporation_id
    JOIN eve_entities AS issuer ON issuer.id = cc.issuer_id
    LEFT JOIN eve_entities AS acceptor ON acceptor.id = cc.acceptor_id
    LEFT JOIN eve_entities AS assignee ON assignee.id = cc.assignee_id
WHERE
    character_id = ?
    AND cc.contract_id = ?;

-- name: ListCharacterContracts :many
SELECT
    sqlc.embed(cc),
    sqlc.embed(issuer_corporation),
    sqlc.embed(issuer),
    acceptor.name as acceptor_name,
    acceptor.category as acceptor_category,
    assignee.name as assignee_name,
    assignee.category as assignee_category
FROM
    character_contracts cc
    JOIN eve_entities AS issuer_corporation ON issuer_corporation.id = cc.issuer_corporation_id
    JOIN eve_entities AS issuer ON issuer.id = cc.issuer_id
    LEFT JOIN eve_entities AS acceptor ON acceptor.id = cc.acceptor_id
    LEFT JOIN eve_entities AS assignee ON assignee.id = cc.assignee_id
WHERE
    character_id = ?;

-- name: ListCharacterContractIDs :many
SELECT
    contract_id
FROM
    character_contracts
WHERE
    character_id = ?;

-- name: UpdateCharacterContract :exec
UPDATE
    character_contracts
SET
    acceptor_id = ?,
    assignee_id = ?,
    date_accepted = ?,
    date_completed = ?,
    status = ?
WHERE
    character_id = ?
    AND contract_id = ?;