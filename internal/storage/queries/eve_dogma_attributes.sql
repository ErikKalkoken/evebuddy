-- name: CreateEveDogmaAttribute :exec
INSERT INTO eve_dogma_attributes (
    id,
    default_value,
    description,
    display_name,
    icon_id,
    name,
    is_high_good,
    is_published,
    is_stackable,
    unit_id
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetEveDogmaAttribute :one
SELECT *
FROM eve_dogma_attributes
WHERE id = ?;
