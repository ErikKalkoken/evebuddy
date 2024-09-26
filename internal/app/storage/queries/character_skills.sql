-- name: GetCharacterSkill :one
SELECT
    sqlc.embed(character_skills),
    sqlc.embed(eve_types),
    sqlc.embed(eve_groups),
    sqlc.embed(eve_categories)
FROM character_skills
JOIN eve_types ON eve_types.id = character_skills.eve_type_id
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id
JOIN eve_categories ON eve_categories.id = eve_groups.eve_category_id
WHERE character_id = ?
AND eve_type_id = ?;

-- name: DeleteCharacterSkills :exec
DELETE FROM character_skills
WHERE character_id = ?
AND eve_type_id IN (sqlc.slice('eve_type_ids'));

-- name: ListCharacterSkillIDs :many
SELECT eve_type_id
FROM character_skills
WHERE character_id = ?;

-- name: ListCharacterShipsAbilities :many
SELECT DISTINCT ss2.ship_type_id as type_id, et.name as type_name, eg.id as group_id, eg.name as group_name,
(
	SELECT COUNT(*) - SUM(IFNULL(cs.active_skill_level, 0) >= ss.skill_level) == 0
	FROM eve_ship_skills ss
	LEFT JOIN character_skills cs ON cs.eve_type_id = ss.skill_type_id AND cs.character_id = ?
	WHERE ss.ship_type_id = ss2.ship_type_id
) as can_fly
FROM eve_ship_skills ss2
JOIN eve_types et ON et.ID = ss2.ship_type_id
JOIN eve_groups eg ON eg.ID = et.eve_group_id
WHERE et.name LIKE ?
ORDER BY et.name;

-- name: ListCharacterShipSkills :many
SELECT
    rank,
    ship_type_id,
    skill_type_id,
    skt.name as skill_name,
    skill_level,
    cs.active_skill_level,
    cs.trained_skill_level
FROM eve_ship_skills ess
JOIN eve_types as sht ON sht.id = ess.ship_type_id
JOIN eve_types as skt ON skt.id = ess.skill_type_id
LEFT JOIN character_skills cs ON cs.eve_type_id = skill_type_id AND cs.character_id = ?
WHERE ship_type_id = ?
ORDER BY RANK;

-- name: ListCharacterSkillGroupsProgress :many
SELECT
    eve_groups.id as eve_group_id,
    eve_groups.name as eve_group_name,
    COUNT(eve_types.id) as total,
    SUM(character_skills.trained_skill_level / 5.0) AS trained
FROM eve_types
JOIN eve_groups ON eve_groups.id = eve_types.eve_group_id AND eve_groups.is_published IS TRUE
LEFT JOIN character_skills ON character_skills.eve_type_id = eve_types.id AND character_skills.character_id = ?
WHERE eve_groups.eve_category_id = ?
AND eve_types.is_published IS TRUE
GROUP BY eve_groups.name
ORDER BY eve_groups.name;

-- name: ListCharacterSkillProgress :many
SELECT
    eve_types.id,
    eve_types.name,
    eve_types.description,
    character_skills.active_skill_level,
    character_skills.trained_skill_level
FROM eve_types
LEFT JOIN character_skills ON character_skills.eve_type_id = eve_types.id AND character_skills.character_id = ?
WHERE eve_types.eve_group_id = ?
AND eve_types.is_published IS TRUE
ORDER BY eve_types.name;

-- name: UpdateOrCreateCharacterSkill :exec
INSERT INTO character_skills (
    character_id,
    eve_type_id,
    active_skill_level,
    skill_points_in_skill,
    trained_skill_level
)
VALUES (
    ?1, ?2, ?3, ?4, ?5
)
ON CONFLICT(character_id, eve_type_id) DO
UPDATE SET
    active_skill_level = ?3,
    skill_points_in_skill = ?4,
    trained_skill_level = ?5
WHERE character_id = ?1
AND eve_type_id = ?2;
