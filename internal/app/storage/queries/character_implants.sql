-- name: CreateCharacterImplant :exec
INSERT INTO character_implants (
    character_id,
    eve_type_id
)
VALUES (
    ?, ?
);

-- name: GetCharacterImplant :one
SELECT
    sqlc.embed(character_implants),
    sqlc.embed(eve_types),
    sqlc.embed(eve_groups),
    sqlc.embed(eve_categories),
    eve_type_dogma_attributes.value as slot_num
FROM character_implants
JOIN eve_types ON eve_types.id = character_implants.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
LEFT JOIN eve_type_dogma_attributes ON eve_type_dogma_attributes.eve_type_id = character_implants.eve_type_id AND eve_type_dogma_attributes.dogma_attribute_id = ?
WHERE character_id = ?
AND character_implants.eve_type_id = ?;

-- name: DeleteCharacterImplants :exec
DELETE FROM character_implants
WHERE character_id = ?;

-- name: ListCharacterImplants :many
SELECT
    sqlc.embed(character_implants),
    sqlc.embed(eve_types),
    sqlc.embed(eve_groups),
    sqlc.embed(eve_categories),
    eve_type_dogma_attributes.value as slot_num
FROM character_implants
JOIN eve_types ON eve_types.id = character_implants.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
LEFT JOIN eve_type_dogma_attributes ON eve_type_dogma_attributes.eve_type_id = character_implants.eve_type_id AND eve_type_dogma_attributes.dogma_attribute_id = ?
WHERE character_id = ?
ORDER BY slot_num;
