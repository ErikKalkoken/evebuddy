-- name: UpdateOrCreateCorporationHangarName :exec
INSERT INTO
    corporation_hangar_names (corporation_id, division_id, name)
VALUES
    (?1, ?2, ?3)
ON CONFLICT (corporation_id, division_id) DO UPDATE
SET
    name = ?3;

-- name: GetCorporationHangarName :one
SELECT
    *
FROM
    corporation_hangar_names
WHERE
    corporation_id = ?
    AND division_id = ?;

-- name: ListCorporationHangarNames :many
SELECT
    *
FROM
    corporation_hangar_names
WHERE
    corporation_id = ?;
