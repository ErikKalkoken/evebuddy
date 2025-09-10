-- name: DeleteCorporationStructures :exec
DELETE FROM corporation_structures
WHERE
    corporation_id = ?
    AND structure_id IN (sqlc.slice('structure_ids'));

-- name: GetCorporationStructure :one
SELECT
    sqlc.embed(cs),
    sqlc.embed(ess),
    sqlc.embed(ecn),
    sqlc.embed(er),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ect)
FROM
    corporation_structures cs
    JOIN eve_solar_systems ess ON ess.ID = cs.system_id
    JOIN eve_constellations ecn ON ecn.ID = ess.eve_constellation_id
    JOIN eve_regions er ON er.ID = ecn.eve_region_id
    JOIN eve_types et ON et.ID = cs.type_id
    JOIN eve_groups eg on eg.id = et.eve_group_id
    JOIN eve_categories ect on ect.id = eg.eve_category_id
WHERE
    corporation_id = ?
    AND structure_id = ?;

-- name: ListCorporationStructures :many
SELECT
    sqlc.embed(cs),
    sqlc.embed(ess),
    sqlc.embed(ecn),
    sqlc.embed(er),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ect)
FROM
    corporation_structures cs
    JOIN eve_solar_systems ess ON ess.ID = cs.system_id
    JOIN eve_constellations ecn ON ecn.ID = ess.eve_constellation_id
    JOIN eve_regions er ON er.ID = ecn.eve_region_id
    JOIN eve_types et ON et.ID = cs.type_id
    JOIN eve_groups eg on eg.id = et.eve_group_id
    JOIN eve_categories ect on ect.id = eg.eve_category_id
WHERE
    corporation_id = ?;

-- name: ListCorporationStructureIDs :many
SELECT
    structure_id
FROM
    corporation_structures
WHERE
    corporation_id = ?;

-- name: UpdateOrCreateCorporationStructure :exec
INSERT INTO
    corporation_structures (
        corporation_id,
        fuel_expires,
        name,
        next_reinforce_apply,
        next_reinforce_hour,
        profile_id,
        reinforce_hour,
        state,
        state_timer_end,
        state_timer_start,
        structure_id,
        system_id,
        type_id,
        unanchors_at
    )
VALUES
    (
        ?1,
        ?2,
        ?3,
        ?4,
        ?5,
        ?6,
        ?7,
        ?8,
        ?9,
        ?10,
        ?11,
        ?12,
        ?13,
        ?14
    )
ON CONFLICT (corporation_id, structure_id) DO UPDATE
SET
    fuel_expires = ?2,
    name = ?3,
    next_reinforce_apply = ?4,
    next_reinforce_hour = ?5,
    reinforce_hour = ?7,
    state = ?8,
    state_timer_end = ?9,
    state_timer_start = ?10,
    unanchors_at = ?14;
