-- name: CreateCorporationMember :exec
INSERT INTO
    corporation_members (corporation_id, character_id)
VALUES
    (?, ?);

-- name: DeleteCorporationMembers :exec
DELETE FROM corporation_members
WHERE
    corporation_id = ?
    AND character_id IN (sqlc.slice('character_ids'));

-- name: GetCorporationMembers :one
SELECT
    sqlc.embed(cm),
    sqlc.embed(ee)
FROM
    corporation_members cm
    JOIN eve_entities ee ON ee.id = cm.character_id
WHERE
    cm.corporation_id = ?
    AND cm.character_id = ?;

-- name: ListCorporationMembers :many
SELECT
    sqlc.embed(cm),
    sqlc.embed(ee)
FROM
    corporation_members cm
    JOIN eve_entities ee ON ee.id = cm.character_id
WHERE
    cm.corporation_id = ?;

-- name: ListCorporationMemberIDs :many
SELECT
    character_id
FROM
    corporation_members
WHERE
    corporation_id = ?;
