-- name: UpdateOrCreateCharacterIndustryJobs :exec
INSERT INTO
    character_industry_jobs (
        activity_id,
        blueprint_id,
        blueprint_location_id,
        blueprint_type_id,
        character_id,
        completed_character_id,
        completed_date,
        cost,
        duration,
        end_date,
        facility_id,
        installer_id,
        job_id,
        licensed_runs,
        output_location_id,
        pause_date,
        probability,
        product_type_id,
        runs,
        start_date,
        station_id,
        status,
        successful_runs
    )
VALUES
    (
        ?1,
        ?2,
        ?3,
        ?4,
        ?5,
        ?6,
        ?7,
        ?8,
        ?9,
        ?10,
        ?11,
        ?12,
        ?13,
        ?14,
        ?15,
        ?16,
        ?17,
        ?18,
        ?19,
        ?20,
        ?21,
        ?22,
        ?23
    )
ON CONFLICT (character_id, job_id) DO UPDATE
SET
    completed_character_id = ?6,
    completed_date = ?7,
    pause_date = ?16,
    status = ?22,
    successful_runs = ?23
WHERE
    character_id = ?5
    AND job_id = ?13;

-- name: GetCharacterIndustryJob :one
SELECT
    sqlc.embed(cij),
    sqlc.embed(ic),
    bl.name AS blueprint_location_name,
    bt.name AS blueprint_type_name,
    cc.name AS completed_character_name,
    ol.name AS output_location_name,
    pt.name AS product_type_name,
    sl.name AS station_name
FROM
    character_industry_jobs cij
    JOIN eve_locations bl ON bl.id = cij.blueprint_location_id
    JOIN eve_types bt ON bt.id = cij.blueprint_type_id
    JOIN eve_entities ic ON ic.id = cij.installer_id
    JOIN eve_locations ol ON ol.id = cij.output_location_id
    JOIN eve_locations sl ON sl.id = cij.station_id
    LEFT JOIN eve_entities cc ON cc.id = cij.completed_character_id
    LEFT JOIN eve_types pt ON pt.id = cij.product_type_id
WHERE
    character_id = ?
    AND job_id = ?;

-- name: ListCharacterIndustryJobs :many
SELECT
    sqlc.embed(cij),
    sqlc.embed(ic),
    bl.name AS blueprint_location_name,
    bt.name AS blueprint_type_name,
    cc.name AS completed_character_name,
    ol.name AS output_location_name,
    pt.name AS product_type_name,
    sl.name AS station_name
FROM
    character_industry_jobs cij
    JOIN eve_locations bl ON bl.id = cij.blueprint_location_id
    JOIN eve_types bt ON bt.id = cij.blueprint_type_id
    JOIN eve_entities ic ON ic.id = cij.installer_id
    JOIN eve_locations ol ON ol.id = cij.output_location_id
    JOIN eve_locations sl ON sl.id = cij.station_id
    LEFT JOIN eve_entities cc ON cc.id = cij.completed_character_id
    LEFT JOIN eve_types pt ON pt.id = cij.product_type_id
WHERE
    character_id = ?
ORDER BY
    start_date DESC;

-- name: ListAllCharacterIndustryJobs :many
SELECT
    sqlc.embed(cij),
    sqlc.embed(ic),
    bl.name AS blueprint_location_name,
    bt.name AS blueprint_type_name,
    cc.name AS completed_character_name,
    ol.name AS output_location_name,
    pt.name AS product_type_name,
    sl.name AS station_name
FROM
    character_industry_jobs cij
    JOIN eve_locations bl ON bl.id = cij.blueprint_location_id
    JOIN eve_types bt ON bt.id = cij.blueprint_type_id
    JOIN eve_entities ic ON ic.id = cij.installer_id
    JOIN eve_locations ol ON ol.id = cij.output_location_id
    JOIN eve_locations sl ON sl.id = cij.station_id
    LEFT JOIN eve_entities cc ON cc.id = cij.completed_character_id
    LEFT JOIN eve_types pt ON pt.id = cij.product_type_id
ORDER BY
    start_date DESC;
