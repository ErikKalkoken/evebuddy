-- name: DeleteCorporationIndustryJobs :exec
DELETE FROM corporation_industry_jobs
WHERE
    corporation_id = ?;

-- name: DeleteCorporationIndustryJobsByID :exec
DELETE FROM corporation_industry_jobs
WHERE
    corporation_id = ?
    AND job_id IN (sqlc.slice('job_ids'));

-- name: GetCorporationIndustryJob :one
SELECT
    sqlc.embed(cij),
    sqlc.embed(ic),
    bt.name AS blueprint_type_name,
    cc.name AS completed_character_name,
    pt.name AS product_type_name,
    lo.name AS location_name,
    los.security_status as station_security
FROM
    corporation_industry_jobs cij
    JOIN eve_types bt ON bt.id = cij.blueprint_type_id
    JOIN eve_entities ic ON ic.id = cij.installer_id
    JOIN eve_locations lo ON lo.id = cij.location_id
    LEFT JOIN eve_solar_systems los ON los.id = lo.eve_solar_system_id
    LEFT JOIN eve_entities cc ON cc.id = cij.completed_character_id
    LEFT JOIN eve_types pt ON pt.id = cij.product_type_id
WHERE
    corporation_id = ?
    AND job_id = ?;

-- name: ListCorporationIndustryJobs :many
SELECT
    sqlc.embed(cij),
    sqlc.embed(ic),
    bt.name AS blueprint_type_name,
    cc.name AS completed_character_name,
    pt.name AS product_type_name,
    lo.name AS location_name,
    los.security_status as station_security
FROM
    corporation_industry_jobs cij
    JOIN eve_types bt ON bt.id = cij.blueprint_type_id
    JOIN eve_entities ic ON ic.id = cij.installer_id
    JOIN eve_locations lo ON lo.id = cij.location_id
    LEFT JOIN eve_solar_systems los ON los.id = lo.eve_solar_system_id
    LEFT JOIN eve_entities cc ON cc.id = cij.completed_character_id
    LEFT JOIN eve_types pt ON pt.id = cij.product_type_id
WHERE
    corporation_id = ?;

-- name: ListAllCorporationIndustryJobs :many
SELECT
    sqlc.embed(cij),
    sqlc.embed(ic),
    bt.name AS blueprint_type_name,
    cc.name AS completed_character_name,
    pt.name AS product_type_name,
    lo.name AS location_name,
    los.security_status as station_security
FROM
    corporation_industry_jobs cij
    JOIN eve_types bt ON bt.id = cij.blueprint_type_id
    JOIN eve_entities ic ON ic.id = cij.installer_id
    JOIN eve_locations lo ON lo.id = cij.location_id
    LEFT JOIN eve_solar_systems los ON los.id = lo.eve_solar_system_id
    LEFT JOIN eve_entities cc ON cc.id = cij.completed_character_id
    LEFT JOIN eve_types pt ON pt.id = cij.product_type_id;

-- name: UpdateCorporationIndustryJobStatus :exec
UPDATE corporation_industry_jobs
SET
    status = ?
WHERE
    corporation_id = ?
    AND job_id IN (sqlc.slice('job_ids'));

-- name: UpdateOrCreateCorporationIndustryJobs :exec
INSERT INTO
    corporation_industry_jobs (
        activity_id,
        blueprint_id,
        blueprint_location_id,
        blueprint_type_id,
        corporation_id,
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
        location_id,
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
ON CONFLICT (corporation_id, job_id) DO UPDATE
SET
    completed_character_id = ?6,
    completed_date = ?7,
    end_date = ?10,
    pause_date = ?16,
    status = ?22,
    successful_runs = ?23;
