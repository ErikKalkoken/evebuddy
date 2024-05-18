-- name: CreateWalletJournalEntry :exec
INSERT INTO wallet_journal_entries (
    amount,
    balance,
    context_id,
    context_id_type,
    date,
    description,
    first_party_id,
    id,
    my_character_id,
    reason,
    ref_type,
    second_party_id,
    tax,
    tax_receiver_id
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetWalletJournalEntry :one
SELECT sqlc.embed(wallet_journal_entries), sqlc.embed(wallet_journal_entry_first_parties), sqlc.embed(wallet_journal_entry_second_parties), sqlc.embed(wallet_journal_entry_tax_receivers)
FROM wallet_journal_entries
LEFT JOIN wallet_journal_entry_first_parties ON wallet_journal_entry_first_parties.id = wallet_journal_entries.first_party_id
LEFT JOIN wallet_journal_entry_second_parties ON wallet_journal_entry_second_parties.id = wallet_journal_entries.second_party_id
LEFT JOIN wallet_journal_entry_tax_receivers ON wallet_journal_entry_tax_receivers.id = wallet_journal_entries.tax_receiver_id
WHERE my_character_id = ? and wallet_journal_entries.id = ?;

-- name: ListWalletJournalEntryIDs :many
SELECT id
FROM wallet_journal_entries
WHERE my_character_id = ?;

-- name: ListWalletJournalEntries :many
SELECT DISTINCT sqlc.embed(wallet_journal_entries), sqlc.embed(wallet_journal_entry_first_parties), sqlc.embed(wallet_journal_entry_second_parties), sqlc.embed(wallet_journal_entry_tax_receivers)
FROM wallet_journal_entries
LEFT JOIN wallet_journal_entry_first_parties ON wallet_journal_entry_first_parties.id = wallet_journal_entries.first_party_id
LEFT JOIN wallet_journal_entry_second_parties ON wallet_journal_entry_second_parties.id = wallet_journal_entries.second_party_id
LEFT JOIN wallet_journal_entry_tax_receivers ON wallet_journal_entry_tax_receivers.id = wallet_journal_entries.tax_receiver_id
WHERE my_character_id = ?
ORDER BY date DESC;
