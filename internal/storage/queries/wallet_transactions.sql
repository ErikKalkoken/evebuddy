-- name: CreateWalletTransaction :exec
INSERT INTO wallet_transactions (
    client_id,
    date,
    eve_type_id,
    is_buy,
    is_personal,
    journal_ref_id,
    my_character_id,
    location_id,
    quantity,
    transaction_id,
    unit_price
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetWalletTransaction :one
SELECT sqlc.embed(wallet_transactions), sqlc.embed(eve_entities), eve_types.name as eve_type_name, locations.name as location_name
FROM wallet_transactions
JOIN eve_entities ON eve_entities.id = wallet_transactions.client_id
JOIN eve_types ON eve_types.id = wallet_transactions.eve_type_id
JOIN locations ON locations.id = wallet_transactions.location_id
WHERE my_character_id = ? and transaction_id = ?;

-- name: ListWalletTransactionIDs :many
SELECT transaction_id
FROM wallet_transactions
WHERE my_character_id = ?;

-- name: ListWalletTransactions :many
SELECT sqlc.embed(wallet_transactions), sqlc.embed(eve_entities), eve_types.name as eve_type_name, locations.name as location_name
FROM wallet_transactions
JOIN eve_entities ON eve_entities.id = wallet_transactions.client_id
JOIN eve_types ON eve_types.id = wallet_transactions.eve_type_id
JOIN locations ON locations.id = wallet_transactions.location_id
WHERE my_character_id = ?
ORDER BY date DESC;
