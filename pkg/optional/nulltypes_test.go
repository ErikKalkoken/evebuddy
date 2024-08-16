package optional_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/pkg/optional"
	"github.com/stretchr/testify/assert"
)

func TestNullTypeConversion(t *testing.T) {
	t.Run("can convert NullInt64 1", func(t *testing.T) {
		x1 := sql.NullInt64{Int64: 42, Valid: true}
		o := optional.FromNullInt64(x1)
		x2 := optional.ToNullInt64(o)
		assert.Equal(t, x1, x2)
	})
	t.Run("can convert NullInt64 2", func(t *testing.T) {
		x1 := sql.NullInt64{}
		o := optional.FromNullInt64(x1)
		x2 := optional.ToNullInt64(o)
		assert.Equal(t, x1, x2)
	})
	t.Run("can convert NullTime 1", func(t *testing.T) {
		x1 := sql.NullTime{Time: time.Now(), Valid: true}
		o := optional.FromNullTime(x1)
		x2 := optional.ToNullTime(o)
		assert.Equal(t, x1, x2)
	})
	t.Run("can convert NullTime 2", func(t *testing.T) {
		x1 := sql.NullTime{}
		o := optional.FromNullTime(x1)
		x2 := optional.ToNullTime(o)
		assert.Equal(t, x1, x2)
	})
	t.Run("can convert NullFloat64 1", func(t *testing.T) {
		x1 := sql.NullFloat64{Float64: 1.23, Valid: true}
		o := optional.FromNullFloat64(x1)
		x2 := optional.ToNullFloat64(o)
		assert.Equal(t, x1, x2)
	})
	t.Run("can convert NullFloat64 2", func(t *testing.T) {
		x1 := sql.NullFloat64{}
		o := optional.FromNullFloat64(x1)
		x2 := optional.ToNullFloat64(o)
		assert.Equal(t, x1, x2)
	})
	t.Run("can convert NullInt64 to int", func(t *testing.T) {
		x1 := sql.NullInt64{Int64: 42, Valid: true}
		o := optional.FromNullInt64ToInteger[int](x1)
		assert.Equal(t, x1.Int64, int64(o.MustValue()))
	})
	t.Run("can convert NullInt64 to int32", func(t *testing.T) {
		x1 := sql.NullInt64{Int64: 42, Valid: true}
		o := optional.FromNullInt64ToInteger[int32](x1)
		assert.Equal(t, x1.Int64, int64(o.MustValue()))
	})
	t.Run("can convert NullInt64 to int 2", func(t *testing.T) {
		x1 := sql.NullInt64{}
		o := optional.FromNullInt64ToInteger[int](x1)
		assert.True(t, o.IsEmpty())
	})
	t.Run("can convert NullString 1", func(t *testing.T) {
		x1 := sql.NullString{String: "alpha", Valid: true}
		o := optional.FromNullString(x1)
		x2 := optional.ToNullString(o)
		assert.Equal(t, x2, x2)
	})
	t.Run("can convert NullString 2", func(t *testing.T) {
		x1 := sql.NullString{}
		o := optional.FromNullString(x1)
		x2 := optional.ToNullString(o)
		assert.Equal(t, x2, x2)
	})
}
