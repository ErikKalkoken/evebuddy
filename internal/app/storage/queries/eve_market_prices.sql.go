// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: eve_market_prices.sql

package queries

import (
	"context"
)

const getEveMarketPrice = `-- name: GetEveMarketPrice :one
SELECT type_id, adjusted_price, average_price
FROM eve_market_prices
WHERE type_id = ?
`

func (q *Queries) GetEveMarketPrice(ctx context.Context, typeID int64) (EveMarketPrice, error) {
	row := q.db.QueryRowContext(ctx, getEveMarketPrice, typeID)
	var i EveMarketPrice
	err := row.Scan(&i.TypeID, &i.AdjustedPrice, &i.AveragePrice)
	return i, err
}

const listEveMarketPrices = `-- name: ListEveMarketPrices :one
SELECT type_id, adjusted_price, average_price
FROM eve_market_prices
`

func (q *Queries) ListEveMarketPrices(ctx context.Context) (EveMarketPrice, error) {
	row := q.db.QueryRowContext(ctx, listEveMarketPrices)
	var i EveMarketPrice
	err := row.Scan(&i.TypeID, &i.AdjustedPrice, &i.AveragePrice)
	return i, err
}

const updateOrCreateEveMarketPrice = `-- name: UpdateOrCreateEveMarketPrice :exec
INSERT INTO eve_market_prices (
    type_id,
    adjusted_price,
    average_price
)
VALUES (
    ?1, ?2, ?3
)
ON CONFLICT(type_id) DO
UPDATE SET
    adjusted_price = ?2,
    average_price = ?3
WHERE type_id = ?1
`

type UpdateOrCreateEveMarketPriceParams struct {
	TypeID        int64
	AdjustedPrice float64
	AveragePrice  float64
}

func (q *Queries) UpdateOrCreateEveMarketPrice(ctx context.Context, arg UpdateOrCreateEveMarketPriceParams) error {
	_, err := q.db.ExecContext(ctx, updateOrCreateEveMarketPrice, arg.TypeID, arg.AdjustedPrice, arg.AveragePrice)
	return err
}