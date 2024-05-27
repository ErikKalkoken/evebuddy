-- name: CreateShipSkill :exec
INSERT INTO ship_skills (
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
FROM ship_skills
JOIN eve_types as ship_types ON ship_types.id = ship_skills.ship_type_id
JOIN eve_types as skill_types ON skill_types.id = ship_skills.skill_type_id
WHERE ship_type_id = ? AND rank = ?;

-- name: ListShipSkills :many
SELECT
    rank,
    ship_type_id,
    skill_type_id,
    skill_level
FROM ship_skills
JOIN eve_types as ship_types ON ship_types.id = ship_skills.ship_type_id
JOIN eve_types as skill_types ON skill_types.id = ship_skills.skill_type_id
WHERE ship_type_id = ?;

-- name: TruncateShipSkills :exec
DELETE FROM ship_skills;


-- name: ListCharacterShipsAbilities :many
SELECT DISTINCT ss2.ship_type_id as type_id, et.name as type_name, eg.id as group_id, eg.name as group_name,
(
	SELECT COUNT(*) - COUNT(NULLIF(0, cs.active_skill_level >= ss.skill_level)) == 0
	FROM ship_skills ss
	LEFT JOIN character_skills cs ON cs.eve_type_id = ss.skill_type_id AND cs.character_id = ?
	WHERE ss.ship_type_id = ss2.ship_type_id
) as can_fly
FROM ship_skills ss2
JOIN eve_types et ON et.ID = ss2.ship_type_id
JOIN eve_groups eg ON eg.ID = et.eve_group_id
WHERE et.name LIKE ?
ORDER BY et.name;