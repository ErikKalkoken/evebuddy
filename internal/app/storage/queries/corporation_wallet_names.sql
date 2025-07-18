-- name: UpdateOrCreateCorporationWalletName :exec
INSERT INTO
    corporation_wallet_names (corporation_id, division_id, name)
VALUES
    (?1, ?2, ?3)
ON CONFLICT (corporation_id, division_id) DO UPDATE
SET
    name = ?3;

-- name: GetCorporationWalletName :one
SELECT
    *
FROM
    corporation_wallet_names
WHERE
    corporation_id = ?
    AND division_id = ?;

-- name: ListCorporationWalletNames :many
SELECT
    *
FROM
    corporation_wallet_names
WHERE
    corporation_id = ?;
