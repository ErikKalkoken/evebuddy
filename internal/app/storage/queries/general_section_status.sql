-- name: GetGeneralSectionStatus :one
SELECT
    *
FROM
    general_section_status
WHERE
    section_id = ?;

-- name: ListGeneralSectionStatus :many
SELECT
    *
FROM
    general_section_status
ORDER BY
    section_id;

-- name: UpdateOrCreateGeneralSectionStatus :one
INSERT INTO
    general_section_status (
        section_id,
        completed_at,
        content_hash,
        error,
        started_at,
        updated_at
    )
VALUES
    (?1, ?2, ?3, ?4, ?5, ?6)
ON CONFLICT (section_id) DO UPDATE
SET
    completed_at = ?2,
    content_hash = ?3,
    error = ?4,
    started_at = ?5,
    updated_at = ?6 RETURNING *;
