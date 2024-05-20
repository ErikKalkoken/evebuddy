-- name: CreateCharacterWalletTransaction :exec
INSERT INTO character_wallet_transactions (
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
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetCharacterWalletTransaction :one
SELECT sqlc.embed(character_wallet_transactions), sqlc.embed(eve_entities), eve_types.name as eve_type_name, locations.name as location_name
FROM character_wallet_transactions
JOIN eve_entities ON eve_entities.id = character_wallet_transactions.client_id
JOIN eve_types ON eve_types.id = character_wallet_transactions.eve_type_id
JOIN locations ON locations.id = character_wallet_transactions.location_id
WHERE character_id = ? and transaction_id = ?;

-- name: ListCharacterWalletTransactionIDs :many
SELECT transaction_id
FROM character_wallet_transactions
WHERE character_id = ?;

-- name: ListCharacterWalletTransactions :many
SELECT sqlc.embed(character_wallet_transactions), sqlc.embed(eve_entities), eve_types.name as eve_type_name, locations.name as location_name
FROM character_wallet_transactions
JOIN eve_entities ON eve_entities.id = character_wallet_transactions.client_id
JOIN eve_types ON eve_types.id = character_wallet_transactions.eve_type_id
JOIN locations ON locations.id = character_wallet_transactions.location_id
WHERE character_id = ?
ORDER BY date DESC;
