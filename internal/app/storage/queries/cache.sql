-- name: CacheClear :exec
DELETE FROM
    cache;

-- name: CacheCleanUp :execrows
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
    AND (
        expires_at > sqlc.arg(now)
        OR expires_at IS NULL
    );

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
    (?1, ?2, ?3) ON CONFLICT(key) DO
UPDATE
SET
    expires_at = ?1,
    value = ?3
WHERE
    key = ?2;