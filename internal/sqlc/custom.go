package sqlc

import (
	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

func Schema() string {
	return schema
}
