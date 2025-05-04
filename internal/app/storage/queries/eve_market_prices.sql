-- name: GetEveMarketPrice :one
SELECT
    *
FROM
    eve_market_prices
WHERE
    type_id = ?;

-- name: ListEveMarketPrices :one
SELECT
    *
FROM
    eve_market_prices;

-- name: UpdateOrCreateEveMarketPrice :exec
INSERT INTO
    eve_market_prices (type_id, adjusted_price, average_price)
VALUES
    (?1, ?2, ?3)
ON CONFLICT (type_id) DO UPDATE
SET
    adjusted_price = ?2,
    average_price = ?3;
