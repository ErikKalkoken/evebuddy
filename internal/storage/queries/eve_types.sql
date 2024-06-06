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
SELECT sqlc.embed(et), sqlc.embed(eg), sqlc.embed(ec)
FROM eve_types et
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE et.id = ?;

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
SELECT sqlc.embed(eda), sqlc.embed(et), sqlc.embed(eg), sqlc.embed(ec), etda.value
FROM eve_dogma_attributes eda
JOIN eve_type_dogma_attributes etda ON etda.dogma_attribute_id = eda.id
join eve_types et ON et.id = etda.eve_type_id
JOIN eve_groups eg ON eg.id = et.eve_group_id
JOIN eve_categories ec ON ec.id = eg.eve_category_id
WHERE etda.eve_type_id = ?;
