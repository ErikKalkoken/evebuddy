-- name: GetCorporationSectionStatus :one
SELECT
    *
FROM
    corporation_section_status
WHERE
    corporation_id = ?
    AND section_id = ?;

-- name: ListCorporationSectionStatus :many
SELECT
    *
FROM
    corporation_section_status
WHERE
    corporation_id = ?;

-- name: UpdateOrCreateCorporationSectionStatus :one
INSERT INTO
    corporation_section_status (
        corporation_id,
        section_id,
        completed_at,
        content_hash,
        error,
        started_at,
        updated_at
    )
VALUES
    (?1, ?2, ?3, ?4, ?5, ?6, ?7)
ON CONFLICT (corporation_id, section_id) DO UPDATE
SET
    completed_at = ?3,
    content_hash = ?4,
    error = ?5,
    started_at = ?6,
    updated_at = ?7
WHERE
    corporation_id = ?1
    AND section_id = ?2 RETURNING *;
