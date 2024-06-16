package sqlite

import (
	"database/sql"
	"time"
)

func NewNullFloat64(v float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: v, Valid: true}
}

func NewNullInt32(v int32) sql.NullInt32 {
	return sql.NullInt32{Int32: v, Valid: true}
}

func NewNullInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: true}
}

func NewNullString(v string) sql.NullString {
	return sql.NullString{String: v, Valid: true}
}

func NewNullTime(v time.Time) sql.NullTime {
	return sql.NullTime{Time: v, Valid: true}
}
