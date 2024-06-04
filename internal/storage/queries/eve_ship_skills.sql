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
    skt.name as skill_name,
    skill_level
FROM eve_ship_skills ess
JOIN eve_types as sht ON sht.id = ess.ship_type_id
JOIN eve_types as skt ON skt.id = ess.skill_type_id
WHERE ship_type_id = ? AND rank = ?;

-- name: ListShipSkills :many
SELECT
    rank,
    ship_type_id,
    skill_type_id,
    skt.name as skill_name,
    skill_level
FROM eve_ship_skills ess
JOIN eve_types as sht ON sht.id = ess.ship_type_id
JOIN eve_types as skt ON skt.id = ess.skill_type_id
WHERE ship_type_id = ?
ORDER BY RANK;

-- name: TruncateShipSkills :exec
DELETE FROM eve_ship_skills;
