-- name: CreateStructureService :exec
INSERT INTO
    corporation_structure_services (corporation_structure_id, name, state)
VALUES
    (?, ?, ?);

-- name: DeleteStructureServices :exec
DELETE FROM corporation_structure_services
WHERE
    corporation_structure_id = ?;

-- name: GetStructureService :one
SELECT
    *
FROM
    corporation_structure_services
WHERE
    corporation_structure_id = ?
    AND name = ?;

-- name: ListStructureServices :many
SELECT
    *
FROM
    corporation_structure_services
WHERE
    corporation_structure_id = ?;