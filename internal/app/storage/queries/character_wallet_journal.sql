-- name: CreateCharacterWalletJournalEntry :exec
INSERT INTO character_wallet_journal_entries (
    amount,
    balance,
    context_id,
    context_id_type,
    date,
    description,
    first_party_id,
    ref_id,
    character_id,
    reason,
    ref_type,
    second_party_id,
    tax,
    tax_receiver_id
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetCharacterWalletJournalEntry :one
SELECT
    sqlc.embed(wje),
    fp.name as first_name,
    fp.category as first_category,
    sp.name as second_name,
    sp.category as second_category,
    tr.name as tax_name,
    tr.category as tax_category
FROM character_wallet_journal_entries wje
LEFT JOIN eve_entities AS fp ON fp.id = wje.first_party_id
LEFT JOIN eve_entities AS sp ON sp.id = wje.second_party_id
LEFT JOIN eve_entities AS tr ON tr.id = wje.tax_receiver_id
WHERE character_id = ? and wje.ref_id = ?;

-- name: ListCharacterWalletJournalEntryRefIDs :many
SELECT ref_id
FROM character_wallet_journal_entries
WHERE character_id = ?;

-- name: ListCharacterWalletJournalEntries :many
SELECT
    sqlc.embed(wje),
    fp.name as first_name,
    fp.category as first_category,
    sp.name as second_name,
    sp.category as second_category,
    tr.name as tax_name,
    tr.category as tax_category
FROM character_wallet_journal_entries wje
LEFT JOIN eve_entities AS fp ON fp.id = wje.first_party_id
LEFT JOIN eve_entities AS sp ON sp.id = wje.second_party_id
LEFT JOIN eve_entities AS tr ON tr.id = wje.tax_receiver_id
WHERE character_id = ?
ORDER BY date DESC;
