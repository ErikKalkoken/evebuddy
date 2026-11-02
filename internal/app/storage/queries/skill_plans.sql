-- name: CreateSkillPlan :one
INSERT INTO skill_plans (name) VALUES (?) RETURNING *;

-- name: UpdateSkillPlan :one
UPDATE skill_plans SET name = ? WHERE id = ? RETURNING *;

-- name: DeleteSkillPlan :execrows
DELETE FROM skill_plans WHERE id = ?;

-- name: ClearSkillPlanRequirements :exec
DELETE FROM skill_plan_requirements WHERE plan_id = ?;

-- name: CreateSkillPlanRequirement :exec
INSERT INTO skill_plan_requirements (plan_id, type_id, level) VALUES (?, ?, ?);

-- name: ListSkillPlans :many
SELECT * FROM skill_plans ORDER BY name COLLATE NOCASE, id;

-- name: ListSkillPlanRequirements :many
SELECT r.plan_id, r.type_id, r.level, et.name
FROM skill_plan_requirements r
JOIN eve_types et ON et.id = r.type_id
ORDER BY et.name COLLATE NOCASE, r.type_id;
