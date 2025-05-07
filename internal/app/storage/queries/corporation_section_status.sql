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
        comment,
        corporation_id,
        section_id,
        completed_at,
        content_hash,
        error,
        started_at,
        updated_at
    )
VALUES
    (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)
ON CONFLICT (corporation_id, section_id) DO UPDATE
SET
    comment = ?1,
    completed_at = ?4,
    content_hash = ?5,
    error = ?6,
    started_at = ?7,
    updated_at = ?8
RETURNING *;
