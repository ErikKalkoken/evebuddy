-- name: CreateScope :one
INSERT INTO scopes (
    name
)
VALUES (
    ?
)
RETURNING *;

-- name: GetScope :one
SELECT *
FROM scopes
WHERE name = ?;
