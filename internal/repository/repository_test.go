package repository_test

import (
	"database/sql"
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/repository"
	"example/evebuddy/internal/sqlc"
)

func setUpDB() (*sql.DB, *sqlc.Queries, factory.Factory) {
	db, err := repository.ConnectDB(":memory:", true)
	if err != nil {
		panic(err)
	}
	q := sqlc.New(db)
	factory := factory.New(q)
	return db, q, factory
}
