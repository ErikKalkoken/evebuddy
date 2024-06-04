-- name: CreateEveType :exec
INSERT INTO eve_types (
    id,
    eve_group_id,
    capacity,
    description,
    graphic_id,
    icon_id,
    is_published,
    market_group_id,
    mass,
    name,
    packaged_volume,
    portion_size,
    radius,
    volume
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetEveType :one
SELECT sqlc.embed(eve_types), sqlc.embed(eve_groups), sqlc.embed(eve_categories)
FROM eve_types
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
WHERE eve_types.id = ?;

-- name: ListEveTypeIDs :many
SELECT id
FROM eve_types;

-- name: CreateEveTypeDogmaAttribute :exec
INSERT INTO eve_type_dogma_attributes (
    dogma_attribute_id,
    eve_type_id,
    value
)
VALUES (
    ?, ?, ?
);

-- name: GetEveTypeDogmaAttribute :one
SELECT *
FROM eve_type_dogma_attributes
WHERE dogma_attribute_id = ?
AND eve_type_id = ?;

-- name: CreateEveTypeDogmaEffect :exec
INSERT INTO eve_type_dogma_effects (
    dogma_effect_id,
    eve_type_id,
    is_default
)
VALUES (
    ?, ?, ?
);

-- name: GetEveTypeDogmaEffect :one
SELECT *
FROM eve_type_dogma_effects
WHERE dogma_effect_id = ?
AND eve_type_id = ?;

-- name: ListEveTypeDogmaAttributesForType :many
SELECT sqlc.embed(eda), etda.eve_type_id, etda.value
FROM eve_dogma_attributes eda
JOIN eve_type_dogma_attributes etda ON etda.dogma_attribute_id = eda.id
WHERE etda.eve_type_id = ?;
