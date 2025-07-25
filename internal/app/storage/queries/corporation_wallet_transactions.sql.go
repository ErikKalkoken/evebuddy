// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: corporation_wallet_transactions.sql

package queries

import (
	"context"
	"database/sql"
	"time"
)

const createCorporationWalletTransaction = `-- name: CreateCorporationWalletTransaction :exec
INSERT INTO
    corporation_wallet_transactions (
        client_id,
        date,
        division_id,
        eve_type_id,
        is_buy,
        journal_ref_id,
        corporation_id,
        location_id,
        quantity,
        transaction_id,
        unit_price
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

type CreateCorporationWalletTransactionParams struct {
	ClientID      int64
	Date          time.Time
	DivisionID    int64
	EveTypeID     int64
	IsBuy         bool
	JournalRefID  int64
	CorporationID int64
	LocationID    int64
	Quantity      int64
	TransactionID int64
	UnitPrice     float64
}

func (q *Queries) CreateCorporationWalletTransaction(ctx context.Context, arg CreateCorporationWalletTransactionParams) error {
	_, err := q.db.ExecContext(ctx, createCorporationWalletTransaction,
		arg.ClientID,
		arg.Date,
		arg.DivisionID,
		arg.EveTypeID,
		arg.IsBuy,
		arg.JournalRefID,
		arg.CorporationID,
		arg.LocationID,
		arg.Quantity,
		arg.TransactionID,
		arg.UnitPrice,
	)
	return err
}

const deleteCorporationWalletTransactions = `-- name: DeleteCorporationWalletTransactions :exec
DELETE FROM corporation_wallet_transactions
WHERE
    corporation_id = ?
    AND division_id = ?
`

type DeleteCorporationWalletTransactionsParams struct {
	CorporationID int64
	DivisionID    int64
}

func (q *Queries) DeleteCorporationWalletTransactions(ctx context.Context, arg DeleteCorporationWalletTransactionsParams) error {
	_, err := q.db.ExecContext(ctx, deleteCorporationWalletTransactions, arg.CorporationID, arg.DivisionID)
	return err
}

const getCorporationWalletTransaction = `-- name: GetCorporationWalletTransaction :one
SELECT
    cwt.id, cwt.corporation_id, cwt.client_id, cwt.date, cwt.division_id, cwt.eve_type_id, cwt.is_buy, cwt.journal_ref_id, cwt.location_id, cwt.quantity, cwt.transaction_id, cwt.unit_price,
    ee.id, ee.category, ee.name,
    et.id, et.eve_group_id, et.capacity, et.description, et.graphic_id, et.icon_id, et.is_published, et.market_group_id, et.mass, et.name, et.packaged_volume, et.portion_size, et.radius, et.volume,
    eg.id, eg.eve_category_id, eg.name, eg.is_published,
    ec.id, ec.name, ec.is_published,
    el.name as location_name,
    ess.security_status as system_security_status,
    er.id as region_id,
    er.name as region_name
FROM
    corporation_wallet_transactions cwt
    JOIN eve_entities ee ON ee.id = cwt.client_id
    JOIN eve_types et ON et.id = cwt.eve_type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
    JOIN eve_locations el ON el.id = cwt.location_id
    LEFT JOIN eve_solar_systems ess ON ess.id = el.eve_solar_system_id
    LEFT JOIN eve_constellations ON eve_constellations.id = ess.eve_constellation_id
    LEFT JOIN eve_regions er ON er.id = eve_constellations.eve_region_id
WHERE
    corporation_id = ?
    AND division_id = ?
    and transaction_id = ?
`

type GetCorporationWalletTransactionParams struct {
	CorporationID int64
	DivisionID    int64
	TransactionID int64
}

type GetCorporationWalletTransactionRow struct {
	CorporationWalletTransaction CorporationWalletTransaction
	EveEntity                    EveEntity
	EveType                      EveType
	EveGroup                     EveGroup
	EveCategory                  EveCategory
	LocationName                 string
	SystemSecurityStatus         sql.NullFloat64
	RegionID                     sql.NullInt64
	RegionName                   sql.NullString
}

func (q *Queries) GetCorporationWalletTransaction(ctx context.Context, arg GetCorporationWalletTransactionParams) (GetCorporationWalletTransactionRow, error) {
	row := q.db.QueryRowContext(ctx, getCorporationWalletTransaction, arg.CorporationID, arg.DivisionID, arg.TransactionID)
	var i GetCorporationWalletTransactionRow
	err := row.Scan(
		&i.CorporationWalletTransaction.ID,
		&i.CorporationWalletTransaction.CorporationID,
		&i.CorporationWalletTransaction.ClientID,
		&i.CorporationWalletTransaction.Date,
		&i.CorporationWalletTransaction.DivisionID,
		&i.CorporationWalletTransaction.EveTypeID,
		&i.CorporationWalletTransaction.IsBuy,
		&i.CorporationWalletTransaction.JournalRefID,
		&i.CorporationWalletTransaction.LocationID,
		&i.CorporationWalletTransaction.Quantity,
		&i.CorporationWalletTransaction.TransactionID,
		&i.CorporationWalletTransaction.UnitPrice,
		&i.EveEntity.ID,
		&i.EveEntity.Category,
		&i.EveEntity.Name,
		&i.EveType.ID,
		&i.EveType.EveGroupID,
		&i.EveType.Capacity,
		&i.EveType.Description,
		&i.EveType.GraphicID,
		&i.EveType.IconID,
		&i.EveType.IsPublished,
		&i.EveType.MarketGroupID,
		&i.EveType.Mass,
		&i.EveType.Name,
		&i.EveType.PackagedVolume,
		&i.EveType.PortionSize,
		&i.EveType.Radius,
		&i.EveType.Volume,
		&i.EveGroup.ID,
		&i.EveGroup.EveCategoryID,
		&i.EveGroup.Name,
		&i.EveGroup.IsPublished,
		&i.EveCategory.ID,
		&i.EveCategory.Name,
		&i.EveCategory.IsPublished,
		&i.LocationName,
		&i.SystemSecurityStatus,
		&i.RegionID,
		&i.RegionName,
	)
	return i, err
}

const listCorporationWalletTransactionIDs = `-- name: ListCorporationWalletTransactionIDs :many
SELECT
    transaction_id
FROM
    corporation_wallet_transactions
WHERE
    corporation_id = ?
    AND division_id = ?
`

type ListCorporationWalletTransactionIDsParams struct {
	CorporationID int64
	DivisionID    int64
}

func (q *Queries) ListCorporationWalletTransactionIDs(ctx context.Context, arg ListCorporationWalletTransactionIDsParams) ([]int64, error) {
	rows, err := q.db.QueryContext(ctx, listCorporationWalletTransactionIDs, arg.CorporationID, arg.DivisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []int64
	for rows.Next() {
		var transaction_id int64
		if err := rows.Scan(&transaction_id); err != nil {
			return nil, err
		}
		items = append(items, transaction_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listCorporationWalletTransactions = `-- name: ListCorporationWalletTransactions :many
SELECT
    cwt.id, cwt.corporation_id, cwt.client_id, cwt.date, cwt.division_id, cwt.eve_type_id, cwt.is_buy, cwt.journal_ref_id, cwt.location_id, cwt.quantity, cwt.transaction_id, cwt.unit_price,
    ee.id, ee.category, ee.name,
    et.id, et.eve_group_id, et.capacity, et.description, et.graphic_id, et.icon_id, et.is_published, et.market_group_id, et.mass, et.name, et.packaged_volume, et.portion_size, et.radius, et.volume,
    eg.id, eg.eve_category_id, eg.name, eg.is_published,
    ec.id, ec.name, ec.is_published,
    el.name as location_name,
    ess.security_status as system_security_status,
    er.id as region_id,
    er.name as region_name
FROM
    corporation_wallet_transactions cwt
    JOIN eve_entities ee ON ee.id = cwt.client_id
    JOIN eve_types et ON et.id = cwt.eve_type_id
    JOIN eve_groups eg ON eg.id = et.eve_group_id
    JOIN eve_categories ec ON ec.id = eg.eve_category_id
    JOIN eve_locations el ON el.id = cwt.location_id
    LEFT JOIN eve_solar_systems ess ON ess.id = el.eve_solar_system_id
    LEFT JOIN eve_constellations ON eve_constellations.id = ess.eve_constellation_id
    LEFT JOIN eve_regions er ON er.id = eve_constellations.eve_region_id
WHERE
    corporation_id = ?
    AND division_id = ?
ORDER BY
    date DESC
`

type ListCorporationWalletTransactionsParams struct {
	CorporationID int64
	DivisionID    int64
}

type ListCorporationWalletTransactionsRow struct {
	CorporationWalletTransaction CorporationWalletTransaction
	EveEntity                    EveEntity
	EveType                      EveType
	EveGroup                     EveGroup
	EveCategory                  EveCategory
	LocationName                 string
	SystemSecurityStatus         sql.NullFloat64
	RegionID                     sql.NullInt64
	RegionName                   sql.NullString
}

func (q *Queries) ListCorporationWalletTransactions(ctx context.Context, arg ListCorporationWalletTransactionsParams) ([]ListCorporationWalletTransactionsRow, error) {
	rows, err := q.db.QueryContext(ctx, listCorporationWalletTransactions, arg.CorporationID, arg.DivisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListCorporationWalletTransactionsRow
	for rows.Next() {
		var i ListCorporationWalletTransactionsRow
		if err := rows.Scan(
			&i.CorporationWalletTransaction.ID,
			&i.CorporationWalletTransaction.CorporationID,
			&i.CorporationWalletTransaction.ClientID,
			&i.CorporationWalletTransaction.Date,
			&i.CorporationWalletTransaction.DivisionID,
			&i.CorporationWalletTransaction.EveTypeID,
			&i.CorporationWalletTransaction.IsBuy,
			&i.CorporationWalletTransaction.JournalRefID,
			&i.CorporationWalletTransaction.LocationID,
			&i.CorporationWalletTransaction.Quantity,
			&i.CorporationWalletTransaction.TransactionID,
			&i.CorporationWalletTransaction.UnitPrice,
			&i.EveEntity.ID,
			&i.EveEntity.Category,
			&i.EveEntity.Name,
			&i.EveType.ID,
			&i.EveType.EveGroupID,
			&i.EveType.Capacity,
			&i.EveType.Description,
			&i.EveType.GraphicID,
			&i.EveType.IconID,
			&i.EveType.IsPublished,
			&i.EveType.MarketGroupID,
			&i.EveType.Mass,
			&i.EveType.Name,
			&i.EveType.PackagedVolume,
			&i.EveType.PortionSize,
			&i.EveType.Radius,
			&i.EveType.Volume,
			&i.EveGroup.ID,
			&i.EveGroup.EveCategoryID,
			&i.EveGroup.Name,
			&i.EveGroup.IsPublished,
			&i.EveCategory.ID,
			&i.EveCategory.Name,
			&i.EveCategory.IsPublished,
			&i.LocationName,
			&i.SystemSecurityStatus,
			&i.RegionID,
			&i.RegionName,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
