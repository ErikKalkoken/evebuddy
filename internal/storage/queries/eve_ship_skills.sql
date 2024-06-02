-- name: CreateShipSkill :exec
INSERT INTO eve_ship_skills (
    rank,
    ship_type_id,
    skill_type_id,
    skill_level
)
VALUES (
    ?, ?, ?, ?
);

-- name: GetShipSkill :one
SELECT
    rank,
    ship_type_id,
    skill_type_id,
    skill_level
FROM eve_ship_skills
JOIN eve_types as ship_types ON ship_types.id = eve_ship_skills.ship_type_id
JOIN eve_types as skill_types ON skill_types.id = eve_ship_skills.skill_type_id
WHERE ship_type_id = ? AND rank = ?;

-- name: ListShipSkills :many
SELECT
    rank,
    ship_type_id,
    skill_type_id,
    skill_level
FROM eve_ship_skills
JOIN eve_types as ship_types ON ship_types.id = eve_ship_skills.ship_type_id
JOIN eve_types as skill_types ON skill_types.id = eve_ship_skills.skill_type_id
WHERE ship_type_id = ?;

-- name: TruncateShipSkills :exec
DELETE FROM eve_ship_skills;
