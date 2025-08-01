-- name: DeleteCharacterIndustryJobs :exec
DELETE FROM character_industry_jobs
WHERE
    character_id = ?
    AND job_id IN (sqlc.slice('job_ids'));

-- name: GetCharacterIndustryJob :one
SELECT
    sqlc.embed(cij),
    sqlc.embed(ic),
    bl.name AS blueprint_location_name,
    bls.security_status as blueprint_location_security,
    bt.name AS blueprint_type_name,
    cc.name AS completed_character_name,
    fc.name AS facility_name,
    fcs.security_status as facility_security,
    ol.name AS output_location_name,
    ols.security_status as output_location_security,
    pt.name AS product_type_name,
    sl.name AS station_name,
    sls.security_status as station_security
FROM
    character_industry_jobs cij
    JOIN eve_locations bl ON bl.id = cij.blueprint_location_id
    JOIN eve_types bt ON bt.id = cij.blueprint_type_id
    JOIN eve_locations fc ON fc.id = cij.facility_id
    JOIN eve_entities ic ON ic.id = cij.installer_id
    JOIN eve_locations ol ON ol.id = cij.output_location_id
    JOIN eve_locations sl ON sl.id = cij.station_id
    LEFT JOIN eve_solar_systems bls ON bls.id = bl.eve_solar_system_id
    LEFT JOIN eve_solar_systems fcs ON fcs.id = fc.eve_solar_system_id
    LEFT JOIN eve_solar_systems sls ON sls.id = sl.eve_solar_system_id
    LEFT JOIN eve_solar_systems ols ON ols.id = ol.eve_solar_system_id
    LEFT JOIN eve_entities cc ON cc.id = cij.completed_character_id
    LEFT JOIN eve_types pt ON pt.id = cij.product_type_id
WHERE
    character_id = ?
    AND job_id = ?;

-- name: ListAllCharacterIndustryJobs :many
SELECT
    sqlc.embed(cij),
    sqlc.embed(ic),
    bl.name AS blueprint_location_name,
    bls.security_status as blueprint_location_security,
    bt.name AS blueprint_type_name,
    cc.name AS completed_character_name,
    fc.name AS facility_name,
    fcs.security_status as facility_security,
    ol.name AS output_location_name,
    ols.security_status as output_location_security,
    pt.name AS product_type_name,
    sl.name AS station_name,
    sls.security_status as station_security
FROM
    character_industry_jobs cij
    JOIN eve_locations bl ON bl.id = cij.blueprint_location_id
    JOIN eve_types bt ON bt.id = cij.blueprint_type_id
    JOIN eve_locations fc ON fc.id = cij.facility_id
    JOIN eve_entities ic ON ic.id = cij.installer_id
    JOIN eve_locations ol ON ol.id = cij.output_location_id
    JOIN eve_locations sl ON sl.id = cij.station_id
    LEFT JOIN eve_solar_systems bls ON bls.id = bl.eve_solar_system_id
    LEFT JOIN eve_solar_systems fcs ON fcs.id = fc.eve_solar_system_id
    LEFT JOIN eve_solar_systems ols ON ols.id = ol.eve_solar_system_id
    LEFT JOIN eve_solar_systems sls ON sls.id = sl.eve_solar_system_id
    LEFT JOIN eve_entities cc ON cc.id = cij.completed_character_id
    LEFT JOIN eve_types pt ON pt.id = cij.product_type_id;

-- name: ListCharacterIndustryJobs :many
SELECT
    sqlc.embed(cij),
    sqlc.embed(ic),
    bl.name AS blueprint_location_name,
    bls.security_status as blueprint_location_security,
    bt.name AS blueprint_type_name,
    cc.name AS completed_character_name,
    fc.name AS facility_name,
    fcs.security_status as facility_security,
    ol.name AS output_location_name,
    ols.security_status as output_location_security,
    pt.name AS product_type_name,
    sl.name AS station_name,
    sls.security_status as station_security
FROM
    character_industry_jobs cij
    JOIN eve_locations bl ON bl.id = cij.blueprint_location_id
    JOIN eve_types bt ON bt.id = cij.blueprint_type_id
    JOIN eve_locations fc ON fc.id = cij.facility_id
    JOIN eve_entities ic ON ic.id = cij.installer_id
    JOIN eve_locations ol ON ol.id = cij.output_location_id
    JOIN eve_locations sl ON sl.id = cij.station_id
    LEFT JOIN eve_solar_systems bls ON bls.id = bl.eve_solar_system_id
    LEFT JOIN eve_solar_systems fcs ON fcs.id = fc.eve_solar_system_id
    LEFT JOIN eve_solar_systems ols ON ols.id = ol.eve_solar_system_id
    LEFT JOIN eve_solar_systems sls ON sls.id = sl.eve_solar_system_id
    LEFT JOIN eve_entities cc ON cc.id = cij.completed_character_id
    LEFT JOIN eve_types pt ON pt.id = cij.product_type_id
WHERE
    character_id = ?;

-- name: ListAllCharacterIndustryJobActiveCounts :many
SELECT
    installer_id,
    activity_id,
    status,
    count(id) as number
FROM
    (
        SELECT
            id,
            installer_id,
            activity_id,
            status
        FROM
            character_industry_jobs j1
        UNION ALL
        SELECT
            j2.id,
            installer_id,
            activity_id,
            status
        FROM
            corporation_industry_jobs j2
            JOIN characters c ON c.id == j2.installer_id
    ) AS jobs
WHERE
    status = "active"
    OR status = "ready"
GROUP BY
    installer_id,
    activity_id,
    status;

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
    end_date = ?10,
    pause_date = ?16,
    status = ?22,
    successful_runs = ?23;
