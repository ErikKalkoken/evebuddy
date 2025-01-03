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
        status_notified,
        title,
        type,
        volume,
        updated_at
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
        ?,
        ?,
        CURRENT_TIMESTAMP
    ) RETURNING id;

-- name: GetCharacterContract :one
SELECT
    sqlc.embed(cc),
    sqlc.embed(issuer_corporation),
    sqlc.embed(issuer),
    acceptor.name as acceptor_name,
    acceptor.category as acceptor_category,
    assignee.name as assignee_name,
    assignee.category as assignee_category,
    end_locations.name as end_location_name,
    start_locations.name as start_location_name,
    end_solar_systems.id as end_solar_system_id,
    end_solar_systems.name as end_solar_system_name,
    start_solar_systems.id as start_solar_system_id,
    start_solar_systems.name as start_solar_system_name,
    (
        SELECT
            IFNULL(GROUP_CONCAT(name || " x " || quantity), "")
        FROM
            character_contract_items cci
            LEFT JOIN eve_types et ON et.id = cci.type_id
        WHERE
            cci.contract_id = cc.id
            AND cci.is_included IS TRUE
    ) as items
FROM
    character_contracts cc
    JOIN eve_entities AS issuer_corporation ON issuer_corporation.id = cc.issuer_corporation_id
    JOIN eve_entities AS issuer ON issuer.id = cc.issuer_id
    LEFT JOIN eve_entities AS acceptor ON acceptor.id = cc.acceptor_id
    LEFT JOIN eve_entities AS assignee ON assignee.id = cc.assignee_id
    LEFT JOIN eve_locations AS end_locations ON end_locations.id = cc.end_location_id
    LEFT JOIN eve_locations AS start_locations ON start_locations.id = cc.start_location_id
    LEFT JOIN eve_solar_systems AS end_solar_systems ON end_solar_systems.id = end_locations.eve_solar_system_id
    LEFT JOIN eve_solar_systems AS start_solar_systems ON start_solar_systems.id = start_locations.eve_solar_system_id
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
    assignee.category as assignee_category,
    end_locations.name as end_location_name,
    start_locations.name as start_location_name,
    end_solar_systems.id as end_solar_system_id,
    end_solar_systems.name as end_solar_system_name,
    start_solar_systems.id as start_solar_system_id,
    start_solar_systems.name as start_solar_system_name,
    (
        SELECT
            IFNULL(GROUP_CONCAT(name || " x " || quantity), "")
        FROM
            character_contract_items cci
            LEFT JOIN eve_types et ON et.id = cci.type_id
        WHERE
            cci.contract_id = cc.id
            AND cci.is_included IS TRUE
    ) as items
FROM
    character_contracts cc
    JOIN eve_entities AS issuer_corporation ON issuer_corporation.id = cc.issuer_corporation_id
    JOIN eve_entities AS issuer ON issuer.id = cc.issuer_id
    LEFT JOIN eve_entities AS acceptor ON acceptor.id = cc.acceptor_id
    LEFT JOIN eve_entities AS assignee ON assignee.id = cc.assignee_id
    LEFT JOIN eve_locations AS end_locations ON end_locations.id = cc.end_location_id
    LEFT JOIN eve_locations AS start_locations ON start_locations.id = cc.start_location_id
    LEFT JOIN eve_solar_systems AS end_solar_systems ON end_solar_systems.id = end_locations.eve_solar_system_id
    LEFT JOIN eve_solar_systems AS start_solar_systems ON start_solar_systems.id = start_locations.eve_solar_system_id
WHERE
    character_id = ?
    AND status <> "deleted";

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
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE
    character_id = ?
    AND contract_id = ?;

-- name: UpdateCharacterContractNotified :exec
UPDATE
    character_contracts
SET
    status_notified = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE
    character_id = ?
    AND contract_id = ?;