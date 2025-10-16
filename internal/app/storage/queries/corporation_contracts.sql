-- name: CreateCorporationContract :one
INSERT INTO
    corporation_contracts (
        acceptor_id,
        assignee_id,
        availability,
        buyout,
        corporation_id,
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
        updated_at,
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
        ?,
        ?,
        ?
    ) RETURNING id;

-- name: GetCorporationContract :one
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
    end_solar_systems.security_status as end_solar_system_security_status,
    start_solar_systems.id as start_solar_system_id,
    start_solar_systems.name as start_solar_system_name,
    start_solar_systems.security_status as start_solar_system_security_status,
    (
        SELECT
            IFNULL(GROUP_CONCAT(name || " x " || quantity), "")
        FROM
            corporation_contract_items cci
            LEFT JOIN eve_types et ON et.id = cci.type_id
        WHERE
            cci.contract_id = cc.id
    ) as items
FROM
    corporation_contracts cc
    JOIN eve_entities AS issuer_corporation ON issuer_corporation.id = cc.issuer_corporation_id
    JOIN eve_entities AS issuer ON issuer.id = cc.issuer_id
    LEFT JOIN eve_entities AS acceptor ON acceptor.id = cc.acceptor_id
    LEFT JOIN eve_entities AS assignee ON assignee.id = cc.assignee_id
    LEFT JOIN eve_locations AS end_locations ON end_locations.id = cc.end_location_id
    LEFT JOIN eve_locations AS start_locations ON start_locations.id = cc.start_location_id
    LEFT JOIN eve_solar_systems AS end_solar_systems ON end_solar_systems.id = end_locations.eve_solar_system_id
    LEFT JOIN eve_solar_systems AS start_solar_systems ON start_solar_systems.id = start_locations.eve_solar_system_id
WHERE
    corporation_id = ?
    AND cc.contract_id = ?;

-- name: ListCorporationContracts :many
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
    end_solar_systems.security_status as end_solar_system_security_status,
    start_solar_systems.id as start_solar_system_id,
    start_solar_systems.name as start_solar_system_name,
    start_solar_systems.security_status as start_solar_system_security_status,
    (
        SELECT
            IFNULL(GROUP_CONCAT(name || " x " || quantity), "")
        FROM
            corporation_contract_items cci
            LEFT JOIN eve_types et ON et.id = cci.type_id
        WHERE
            cci.contract_id = cc.id
    ) as items
FROM
    corporation_contracts cc
    JOIN eve_entities AS issuer_corporation ON issuer_corporation.id = cc.issuer_corporation_id
    JOIN eve_entities AS issuer ON issuer.id = cc.issuer_id
    LEFT JOIN eve_entities AS acceptor ON acceptor.id = cc.acceptor_id
    LEFT JOIN eve_entities AS assignee ON assignee.id = cc.assignee_id
    LEFT JOIN eve_locations AS end_locations ON end_locations.id = cc.end_location_id
    LEFT JOIN eve_locations AS start_locations ON start_locations.id = cc.start_location_id
    LEFT JOIN eve_solar_systems AS end_solar_systems ON end_solar_systems.id = end_locations.eve_solar_system_id
    LEFT JOIN eve_solar_systems AS start_solar_systems ON start_solar_systems.id = start_locations.eve_solar_system_id
WHERE
    corporation_id = ?
GROUP BY
    corporation_id, contract_id
ORDER BY
    date_issued DESC;

-- name: ListCorporationContractIDs :many
SELECT
    contract_id
FROM
    corporation_contracts
WHERE
    corporation_id = ?;

-- name: UpdateCorporationContract :exec
UPDATE
    corporation_contracts
SET
    acceptor_id = ?,
    date_accepted = ?,
    date_completed = ?,
    status = ?,
    updated_at = ?
WHERE
    corporation_id = ?
    AND contract_id = ?;

-- name: UpdateCorporationContractNotified :exec
UPDATE
    corporation_contracts
SET
    status_notified = ?,
    updated_at = ?
WHERE
    id = ?;

-- name: CreateCorporationContractBid :exec
INSERT INTO
    corporation_contract_bids (
        contract_id,
        amount,
        bid_id,
        bidder_id,
        date_bid
    )
VALUES
    (
        ?,
        ?,
        ?,
        ?,
        ?
    );

-- name: GetCorporationContractBid :one
SELECT
    sqlc.embed(ccb),
    sqlc.embed(ee)
FROM
    corporation_contract_bids ccb
    JOIN eve_entities ee ON ee.id = ccb.bidder_id
WHERE
    contract_id = ?
    AND bid_id = ?;

-- name: ListCorporationContractBids :many
SELECT
    sqlc.embed(ccb),
    sqlc.embed(ee)
FROM
    corporation_contract_bids ccb
    JOIN eve_entities ee ON ee.id = ccb.bidder_id
WHERE
    contract_id = ?;

-- name: ListCorporationContractBidIDs :many
SELECT
    bid_id
FROM
    corporation_contract_bids
WHERE
    contract_id = ?;

-- name: CreateCorporationContractItem :exec
INSERT INTO
    corporation_contract_items (
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

-- name: GetCorporationContractItem :one
SELECT
    sqlc.embed(cci),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM
    corporation_contract_items cci
    JOIN eve_types et ON et.id = cci.type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE
    contract_id = ?
    AND record_id = ?;

-- name: ListCorporationContractItems :many
SELECT
    sqlc.embed(cci),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ec)
FROM
    corporation_contract_items cci
    JOIN eve_types et ON et.id = cci.type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE
    contract_id = ?;