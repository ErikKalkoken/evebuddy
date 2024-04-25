-- name: CreateEveType :exec
INSERT INTO eve_types (
    id,
    description,
    eve_group_id,
    name,
    is_published
)
VALUES (
    ?, ?, ?, ?, ?
);

-- name: GetEveType :one
SELECT sqlc.embed(eve_types), sqlc.embed(eve_groups), sqlc.embed(eve_categories)
FROM eve_types
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
WHERE eve_types.id = ?;
