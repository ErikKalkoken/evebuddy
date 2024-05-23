-- name: CreateCharacterJumpClone :one
INSERT INTO character_jump_clones (
    character_id,
    jump_clone_id,
    location_id,
    name
)
VALUES (
    ?, ?, ?, ?
)
RETURNING id;

-- name: DeleteCharacterJumpClones :exec
DELETE FROM character_jump_clones
WHERE character_id = ?;

-- name: GetCharacterJumpClone :one
SELECT sqlc.embed(character_jump_clones), locations.name as location_name
FROM character_jump_clones
JOIN locations ON locations.id = character_jump_clones.location_id
WHERE character_id = ?
AND jump_clone_id = ?;

-- name: ListCharacterJumpClones :many
SELECT DISTINCT
    sqlc.embed(character_jump_clones),
    locations.name as location_name,
    (
        SELECT COUNT(*)
        FROM character_jump_clone_implants
        WHERE clone_id = character_jump_clones.id
    ) AS implants_count
FROM character_jump_clones
JOIN locations ON locations.id = character_jump_clones.location_id
LEFT JOIN character_jump_clone_implants ON character_jump_clone_implants.clone_id = character_jump_clones.id
WHERE character_id = ?
ORDER BY location_name, implants_count DESC;

-- name: CreateCharacterJumpCloneImplant :exec
INSERT INTO character_jump_clone_implants (
    clone_id,
    eve_type_id
)
VALUES (
    ?, ?
);

-- name: ListCharacterJumpCloneImplant :many
SELECT DISTINCT
    sqlc.embed(character_jump_clone_implants),
    sqlc.embed(eve_types),
    sqlc.embed(eve_groups),
    sqlc.embed(eve_categories),
    eve_type_dogma_attributes.value as slot_num
FROM character_jump_clone_implants
JOIN eve_types ON eve_types.id = character_jump_clone_implants.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
LEFT JOIN eve_type_dogma_attributes ON eve_type_dogma_attributes.eve_type_id = character_jump_clone_implants.eve_type_id AND eve_type_dogma_attributes.dogma_attribute_id = ?
WHERE clone_id = ?
ORDER BY slot_num;
