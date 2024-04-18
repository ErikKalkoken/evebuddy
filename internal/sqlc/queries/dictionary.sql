-- name: CreateDictEntry :exec
INSERT INTO dictionary (
    value,
    key
)
VALUES (?, ?);

-- name: DeleteDictEntry :exec
DELETE FROM dictionary
WHERE key = ?;

-- name: GetDictEntry :one
SELECT *
FROM dictionary
WHERE key = ?;

-- name: UpdateDictEntry :exec
UPDATE dictionary
SET value = ?
WHERE key = ?;
