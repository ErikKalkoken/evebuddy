-- name: CreateSetting :exec
INSERT INTO settings (
    value,
    key
)
VALUES (?, ?);

-- name: DeleteSetting :exec
DELETE FROM settings
WHERE key = ?;

-- name: GetSetting :one
SELECT *
FROM settings
WHERE key = ?;

-- name: UpdateSetting :exec
UPDATE settings
SET value = ?
WHERE key = ?;
