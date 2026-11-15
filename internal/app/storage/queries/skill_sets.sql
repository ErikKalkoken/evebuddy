-- name: CreateSkillSet :one
INSERT INTO skill_sets (name) VALUES (?) RETURNING *;

-- name: UpdateSkillSet :one
UPDATE skill_sets SET name = ? WHERE id = ? RETURNING *;

-- name: DeleteSkillSet :execrows
DELETE FROM skill_sets WHERE id = ?;

-- name: ClearSkillSetRequirements :exec
DELETE FROM skill_set_requirements WHERE skill_set_id = ?;

-- name: CreateSkillSetRequirement :exec
INSERT INTO skill_set_requirements (skill_set_id, type_id, level) VALUES (?, ?, ?);

-- name: ListSkillSets :many
SELECT * FROM skill_sets ORDER BY name COLLATE NOCASE, id;

-- name: ListSkillSetRequirements :many
SELECT r.skill_set_id, r.type_id, r.level, et.name
FROM skill_set_requirements r
JOIN eve_types et ON et.id = r.type_id
ORDER BY et.name COLLATE NOCASE, r.type_id;
