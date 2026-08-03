package storage

import (
	"database/sql"
	"time"

	"golang.org/x/exp/constraints"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type nullEntity struct {
	id   sql.NullInt64
	name sql.NullString
}

func entityShortFromNullableDBModel(ne nullEntity) optional.Optional[*app.EntityShort] {
	var o optional.Optional[*app.EntityShort]
	if ne.id.Valid && ne.name.Valid {
		o.Set(&app.EntityShort{
			ID:   ne.id.Int64,
			Name: ne.name.String,
		})
	}
	return o
}

// NewNullFloat64 returns a value as null type. Will assume not set when value is zero.
func NewNullFloat64(v float64) sql.NullFloat64 {
	if v == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: v, Valid: true}
}

// NewNullInt64 returns a value as null type. Will assume not set when value is zero.
func NewNullInt64[T constraints.Integer](v T) sql.NullInt64 {
	if v == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(v), Valid: true}
}

func NewNullString(v string) sql.NullString {
	return sql.NullString{String: v, Valid: true}
}

// NewNullTimeFromTime returns a value as null type. Will assume not set when value is zero.
func NewNullTimeFromTime(v time.Time) sql.NullTime {
	if v.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: v, Valid: true}
}

func NewTimeFromNullTime(v sql.NullTime) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time
}
