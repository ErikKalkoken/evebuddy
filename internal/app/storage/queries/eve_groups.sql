-- name: CreateEveGroup :exec
INSERT INTO
    eve_groups (id, eve_category_id, name, is_published)
VALUES
    (?, ?, ?, ?);

-- name: GetEveGroup :one
SELECT
    sqlc.embed(eve_groups),
    sqlc.embed(eve_categories)
FROM
    eve_groups
    JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
WHERE
    eve_groups.id = ?;

-- name: ListEveGroupsForCategory :many
SELECT
    sqlc.embed(eve_groups),
    sqlc.embed(eve_categories)
FROM
    eve_groups
    JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
WHERE
    eve_groups.eve_category_id = ?;

-- name: ListEveSkillGroups :many
SELECT
    eve_groups.id as eve_group_id,
    eve_groups.name as eve_group_name,
    COUNT(eve_types.id) as skill_count
FROM
    eve_types
    JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
    AND eve_groups.is_published IS TRUE
WHERE
    eve_groups.eve_category_id = ?
    AND eve_types.is_published IS TRUE
GROUP BY
    eve_groups.name;