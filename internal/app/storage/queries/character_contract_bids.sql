-- name: CreateCharacterContractBid :exec
INSERT INTO
    character_contract_bids (
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

-- name: GetCharacterContractBid :one
SELECT
    sqlc.embed(ccb),
    sqlc.embed(ee)
FROM
    character_contract_bids ccb
    JOIN eve_entities ee ON ee.id = ccb.bidder_id
WHERE
    contract_id = ?
    AND bid_id = ?;

-- name: ListCharacterContractBids :many
SELECT
    sqlc.embed(ccb),
    sqlc.embed(ee)
FROM
    character_contract_bids ccb
    JOIN eve_entities ee ON ee.id = ccb.bidder_id
WHERE
    contract_id = ?;

-- name: ListCharacterContractBidIDs :many
SELECT
    bid_id
FROM
    character_contract_bids
WHERE
    contract_id = ?;