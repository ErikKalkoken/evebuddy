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

-- name: DeleteExcludedCharacterSkills :exec
DELETE FROM character_skills
WHERE character_id = ?
AND eve_type_id NOT IN (sqlc.slice('eve_type_ids'));

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
