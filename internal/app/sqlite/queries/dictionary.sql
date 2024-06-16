-- name: DeleteDictEntry :exec
DELETE FROM dictionary
WHERE key = ?;

-- name: GetDictEntry :one
SELECT *
FROM dictionary
WHERE key = ?;

-- name: UpdateOrCreateDictEntry :exec
INSERT INTO dictionary (
    key,
    value
)
VALUES (?1, ?2)
ON CONFLICT(key) DO
UPDATE SET value = ?2
WHERE key = ?1;
