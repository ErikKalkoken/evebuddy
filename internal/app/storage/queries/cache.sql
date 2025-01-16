-- name: CacheClear :exec
DELETE FROM
    cache;

-- name: CacheCleanUp :exec
DELETE FROM
    cache
WHERE
    expires_at < sqlc.arg(now);

-- name: CacheGet :one
SELECT
    *
FROM
    cache
WHERE
    key = ?
    AND expires_at > sqlc.arg(now);

-- name: CacheDelete :exec
DELETE FROM
    cache
WHERE
    key = ?;

-- name: CacheSet :exec
INSERT INTO
    cache (
        expires_at,
        key,
        value
    )
VALUES
    (?, ?, ?);