-- name: UpdateOrCreateCorporationWalletBalance :exec
INSERT INTO
    corporation_wallet_balances (corporation_id, division_id, balance)
VALUES
    (?1, ?2, ?3)
ON CONFLICT (corporation_id, division_id) DO UPDATE
SET
    balance = ?3;

-- name: GetCorporationWalletBalance :one
SELECT
    *
FROM
    corporation_wallet_balances
WHERE
    corporation_id = ?
    AND division_id = ?;

-- name: ListCorporationWalletBalances :many
SELECT
    *
FROM
    corporation_wallet_balances
WHERE
    corporation_id = ?;
