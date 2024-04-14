-- name: DeleteSetting :exec
DELETE FROM settings
WHERE key = ?;

-- name: GetSetting :one
SELECT *
FROM settings
WHERE key = ?;

-- name: UpdateOrCreateSetting :exec
INSERT INTO settings (
    value,
    key
)
VALUES (?, ?)
ON CONFLICT (key) DO
UPDATE SET value = ?;
