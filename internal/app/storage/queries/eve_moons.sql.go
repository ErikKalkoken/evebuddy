// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: eve_moons.sql

package queries

import (
	"context"
)

const createEveMoon = `-- name: CreateEveMoon :exec
INSERT INTO eve_moons (
    id,
    name,
    eve_solar_system_id
)
VALUES (
    ?, ?, ?
)
`

type CreateEveMoonParams struct {
	ID               int64
	Name             string
	EveSolarSystemID int64
}

func (q *Queries) CreateEveMoon(ctx context.Context, arg CreateEveMoonParams) error {
	_, err := q.db.ExecContext(ctx, createEveMoon, arg.ID, arg.Name, arg.EveSolarSystemID)
	return err
}

const getEveMoon = `-- name: GetEveMoon :one
SELECT em.id, em.name, em.eve_solar_system_id, ess.id, ess.eve_constellation_id, ess.name, ess.security_status, ecn.id, ecn.eve_region_id, ecn.name, er.id, er.description, er.name
FROM eve_moons em
JOIN eve_solar_systems ess ON ess.id = em.eve_solar_system_id
JOIN eve_constellations ecn ON ecn.id = ess.eve_constellation_id
JOIN eve_regions er ON er.id = ecn.eve_region_id
WHERE em.id = ?
`

type GetEveMoonRow struct {
	EveMoon          EveMoon
	EveSolarSystem   EveSolarSystem
	EveConstellation EveConstellation
	EveRegion        EveRegion
}

func (q *Queries) GetEveMoon(ctx context.Context, id int64) (GetEveMoonRow, error) {
	row := q.db.QueryRowContext(ctx, getEveMoon, id)
	var i GetEveMoonRow
	err := row.Scan(
		&i.EveMoon.ID,
		&i.EveMoon.Name,
		&i.EveMoon.EveSolarSystemID,
		&i.EveSolarSystem.ID,
		&i.EveSolarSystem.EveConstellationID,
		&i.EveSolarSystem.Name,
		&i.EveSolarSystem.SecurityStatus,
		&i.EveConstellation.ID,
		&i.EveConstellation.EveRegionID,
		&i.EveConstellation.Name,
		&i.EveRegion.ID,
		&i.EveRegion.Description,
		&i.EveRegion.Name,
	)
	return i, err
}
