-- name: CreateCharacterPlanet :one
INSERT INTO
    character_planets (
        character_id,
        eve_planet_id,
        last_notified,
        last_update,
        upgrade_level
    )
VALUES
    (?1, ?2, ?3, ?4, ?5) RETURNING id;

-- name: DeleteCharacterPlanets :exec
DELETE FROM
    character_planets
WHERE
    character_id = ?
    AND eve_planet_id IN (sqlc.slice('eve_planet_ids'));

-- name: GetCharacterPlanet :one
SELECT
    sqlc.embed(cp),
    sqlc.embed(ep),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ect),
    sqlc.embed(ess),
    sqlc.embed(ecs),
    sqlc.embed(er)
FROM
    character_planets cp
    JOIN eve_planets ep ON ep.id = cp.eve_planet_id
    JOIN eve_types et ON et.id = ep.eve_type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ect ON ect.id = eg.eve_category_id
    JOIN eve_solar_systems ess ON ess.id = ep.eve_solar_system_id
    JOIN eve_constellations ecs ON ecs.id = ess.eve_constellation_id
    JOIN eve_regions er ON er.id = ecs.eve_region_id
WHERE
    character_id = ?
    AND eve_planet_id = ?;

-- name: ListCharacterPlanets :many
SELECT
    sqlc.embed(cp),
    sqlc.embed(ep),
    sqlc.embed(et),
    sqlc.embed(eg),
    sqlc.embed(ect),
    sqlc.embed(ess),
    sqlc.embed(ecs),
    sqlc.embed(er)
FROM
    character_planets cp
    JOIN eve_planets ep ON ep.id = cp.eve_planet_id
    JOIN eve_types et ON et.id = ep.eve_type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ect ON ect.id = eg.eve_category_id
    JOIN eve_solar_systems ess ON ess.id = ep.eve_solar_system_id
    JOIN eve_constellations ecs ON ecs.id = ess.eve_constellation_id
    JOIN eve_regions er ON er.id = ecs.eve_region_id
WHERE
    character_id = ?
ORDER BY
    ep.name;

-- name: UpdateCharacterPlanetLastNotified :exec
UPDATE
    character_planets
SET
    last_notified = ?
WHERE
    character_id = ?
    AND eve_planet_id = ?;

-- name: UpdateOrCreateCharacterPlanet :one
INSERT INTO
    character_planets (
        character_id,
        eve_planet_id,
        last_update,
        upgrade_level
    )
VALUES
    (?1, ?2, ?3, ?4) ON CONFLICT(character_id, eve_planet_id) DO
UPDATE
SET
    last_update = ?3,
    upgrade_level = ?4
WHERE
    character_id = ?1
    AND eve_planet_id = ?2 RETURNING id;