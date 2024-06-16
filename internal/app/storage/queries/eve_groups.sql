-- name: CreateEveGroup :exec
INSERT INTO eve_groups (
    id,
    eve_category_id,
    name,
    is_published
)
VALUES (
    ?, ?, ?, ?
);

-- name: GetEveGroup :one
SELECT sqlc.embed(eve_groups), sqlc.embed(eve_categories)
FROM eve_groups
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
WHERE eve_groups.id = ?;
