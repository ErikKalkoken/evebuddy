-- name: CreateCharacterJumpClone :one
INSERT INTO
    character_jump_clones (character_id, jump_clone_id, location_id, name)
VALUES
    (?, ?, ?, ?) RETURNING id;

-- name: DeleteCharacterJumpClones :exec
DELETE FROM character_jump_clones
WHERE
    character_id = ?;

-- name: GetCharacterJumpClone :one
SELECT
    sqlc.embed(cjc),
    el.name as location_name,
    er.id as region_id,
    er.name as region_name,
    ess.security_status as location_security
FROM
    character_jump_clones cjc
    JOIN eve_locations el ON el.id = cjc.location_id
    LEFT JOIN eve_solar_systems ess ON ess.id = el.eve_solar_system_id
    LEFT JOIN eve_constellations ON eve_constellations.id = ess.eve_constellation_id
    LEFT JOIN eve_regions er ON er.id = eve_constellations.eve_region_id
WHERE
    character_id = ?
    AND jump_clone_id = ?;

-- name: ListCharacterJumpClones :many
SELECT DISTINCT
    sqlc.embed(cjc),
    el.name as location_name,
    er.id as region_id,
    er.name as region_name,
    ess.security_status as location_security
FROM
    character_jump_clones cjc
    JOIN eve_locations el ON el.id = cjc.location_id
    LEFT JOIN eve_solar_systems ess ON ess.id = el.eve_solar_system_id
    LEFT JOIN eve_constellations ON eve_constellations.id = ess.eve_constellation_id
    LEFT JOIN eve_regions er ON er.id = eve_constellations.eve_region_id
WHERE
    character_id = ?
ORDER BY
    location_name;

-- name: ListAllCharacterJumpClones :many
SELECT DISTINCT
    cjc.id,
    cjc.character_id,
    ec.name as character_name,
    cjc.location_id,
    cjc.jump_clone_id,
    el.eve_solar_system_id as location_solar_system_id,
    el.eve_type_id as location_type_id,
    el.name as location_name,
    el.owner_id as location_owner_id,
    (
        SELECT
            COUNT(*)
        FROM
            character_jump_clone_implants
        WHERE
            clone_id = cjc.id
    ) AS implants_count
FROM
    character_jump_clones cjc
    JOIN eve_locations el ON el.id = cjc.location_id
    JOIN eve_characters ec ON ec.id = cjc.character_id
    LEFT JOIN character_jump_clone_implants cjci ON cjci.clone_id = cjc.id;

-- name: CreateCharacterJumpCloneImplant :exec
INSERT INTO
    character_jump_clone_implants (clone_id, eve_type_id)
VALUES
    (?, ?);

-- name: ListCharacterJumpCloneImplant :many
SELECT DISTINCT
    sqlc.embed(character_jump_clone_implants),
    sqlc.embed(eve_types),
    sqlc.embed(eve_groups),
    sqlc.embed(eve_categories),
    eve_type_dogma_attributes.value as slot_num
FROM
    character_jump_clone_implants
    JOIN eve_types ON eve_types.id = character_jump_clone_implants.eve_type_id
    JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
    JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
    LEFT JOIN eve_type_dogma_attributes ON eve_type_dogma_attributes.eve_type_id = character_jump_clone_implants.eve_type_id
    AND eve_type_dogma_attributes.dogma_attribute_id = ?
WHERE
    clone_id = ?
ORDER BY
    slot_num;