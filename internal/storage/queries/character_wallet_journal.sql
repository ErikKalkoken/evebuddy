-- name: CreateCharacterWalletJournalEntry :exec
INSERT INTO character_wallet_journal_entries (
    amount,
    balance,
    context_id,
    context_id_type,
    date,
    description,
    first_party_id,
    id,
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
SELECT sqlc.embed(character_wallet_journal_entries), sqlc.embed(character_wallet_journal_entry_first_parties), sqlc.embed(character_wallet_journal_entry_second_parties), sqlc.embed(character_wallet_journal_entry_tax_receivers)
FROM character_wallet_journal_entries
LEFT JOIN character_wallet_journal_entry_first_parties ON character_wallet_journal_entry_first_parties.id = character_wallet_journal_entries.first_party_id
LEFT JOIN character_wallet_journal_entry_second_parties ON character_wallet_journal_entry_second_parties.id = character_wallet_journal_entries.second_party_id
LEFT JOIN character_wallet_journal_entry_tax_receivers ON character_wallet_journal_entry_tax_receivers.id = character_wallet_journal_entries.tax_receiver_id
WHERE character_id = ? and character_wallet_journal_entries.id = ?;

-- name: ListCharacterWalletJournalEntryIDs :many
SELECT id
FROM character_wallet_journal_entries
WHERE character_id = ?;

-- name: ListCharacterWalletJournalEntries :many
SELECT DISTINCT sqlc.embed(character_wallet_journal_entries), sqlc.embed(character_wallet_journal_entry_first_parties), sqlc.embed(character_wallet_journal_entry_second_parties), sqlc.embed(character_wallet_journal_entry_tax_receivers)
FROM character_wallet_journal_entries
LEFT JOIN character_wallet_journal_entry_first_parties ON character_wallet_journal_entry_first_parties.id = character_wallet_journal_entries.first_party_id
LEFT JOIN character_wallet_journal_entry_second_parties ON character_wallet_journal_entry_second_parties.id = character_wallet_journal_entries.second_party_id
LEFT JOIN character_wallet_journal_entry_tax_receivers ON character_wallet_journal_entry_tax_receivers.id = character_wallet_journal_entries.tax_receiver_id
WHERE character_id = ?
ORDER BY date DESC;
