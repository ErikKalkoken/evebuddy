-- name: CreateCorporationWalletJournalEntry :exec
INSERT INTO
    corporation_wallet_journal_entries (
        amount,
        balance,
        context_id,
        context_id_type,
        date,
        description,
        division_id,
        first_party_id,
        ref_id,
        corporation_id,
        reason,
        ref_type,
        second_party_id,
        tax,
        tax_receiver_id
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetCorporationWalletJournalEntry :one
SELECT
    sqlc.embed(wje),
    fp.name as first_name,
    fp.category as first_category,
    sp.name as second_name,
    sp.category as second_category,
    tr.name as tax_name,
    tr.category as tax_category
FROM
    corporation_wallet_journal_entries wje
    LEFT JOIN eve_entities AS fp ON fp.id = wje.first_party_id
    LEFT JOIN eve_entities AS sp ON sp.id = wje.second_party_id
    LEFT JOIN eve_entities AS tr ON tr.id = wje.tax_receiver_id
WHERE
    corporation_id = ?
    AND wje.ref_id = ?
    and wje.division_id = ?;

-- name: ListCorporationWalletJournalEntryRefIDs :many
SELECT
    ref_id
FROM
    corporation_wallet_journal_entries
WHERE
    corporation_id = ?
    AND division_id = ?;

-- name: ListCorporationWalletJournalEntries :many
SELECT
    sqlc.embed(wje),
    fp.name as first_name,
    fp.category as first_category,
    sp.name as second_name,
    sp.category as second_category,
    tr.name as tax_name,
    tr.category as tax_category
FROM
    corporation_wallet_journal_entries wje
    LEFT JOIN eve_entities AS fp ON fp.id = wje.first_party_id
    LEFT JOIN eve_entities AS sp ON sp.id = wje.second_party_id
    LEFT JOIN eve_entities AS tr ON tr.id = wje.tax_receiver_id
WHERE
    corporation_id = ?
    AND division_id = ?
ORDER BY
    date DESC;
