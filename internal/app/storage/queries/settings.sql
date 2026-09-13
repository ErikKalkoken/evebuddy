-- name: GetSetting :one
SELECT
    value
FROM
    settings
WHERE
    key = ?;

-- name: ListSettings :many
SELECT
    key,
    value
FROM
    settings;

-- name: DeleteSetting :exec
DELETE FROM settings
WHERE
    key = ?;

-- name: SetSetting :exec
INSERT INTO
    settings (key, value)
VALUES
    (?1, ?2)
ON CONFLICT (key) DO UPDATE
SET
    value = ?2
WHERE
    key = ?1;