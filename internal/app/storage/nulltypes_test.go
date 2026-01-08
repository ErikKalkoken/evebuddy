package storage_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func TestNullTypes(t *testing.T) {
	t.Run("should convert float64 from value", func(t *testing.T) {
		x := storage.NewNullFloat64(1.2)
		assert.Equal(t, sql.NullFloat64{Float64: 1.2, Valid: true}, x)
	})
	t.Run("should convert float64 from zero", func(t *testing.T) {
		x := storage.NewNullFloat64(0)
		assert.Equal(t, sql.NullFloat64{}, x)
	})
	t.Run("should convert int64 from value", func(t *testing.T) {
		x := storage.NewNullInt64(42)
		assert.Equal(t, sql.NullInt64{Int64: 42, Valid: true}, x)
	})
	t.Run("should convert int64 from zero", func(t *testing.T) {
		x := storage.NewNullInt64(0)
		assert.Equal(t, sql.NullInt64{Int64: 0, Valid: false}, x)
	})
	t.Run("should convert string", func(t *testing.T) {
		x := storage.NewNullString("alpha")
		assert.Equal(t, sql.NullString{String: "alpha", Valid: true}, x)
	})
}

func TestNullTimeFromTime(t *testing.T) {
	t.Run("can convert normal time", func(t *testing.T) {
		t1 := time.Now()
		x := storage.NewNullTimeFromTime(t1)
		assert.Equal(t, sql.NullTime{Time: t1, Valid: true}, x)
	})
	t.Run("should convert zero time into null", func(t *testing.T) {
		t1 := time.Time{}
		x := storage.NewNullTimeFromTime(t1)
		assert.False(t, x.Valid)
	})
}

func TestTimeFromNullTime(t *testing.T) {
	t.Run("can convert valid time", func(t *testing.T) {
		t1 := time.Now()
		t2 := sql.NullTime{Time: t1, Valid: true}
		x := storage.NewTimeFromNullTime(t2)
		assert.Equal(t, t1, x)
	})
	t.Run("should convert null time into zero time", func(t *testing.T) {
		t2 := sql.NullTime{Time: time.Time{}, Valid: false}
		x := storage.NewTimeFromNullTime(t2)
		assert.True(t, x.IsZero())
	})
}

func TestIntegerFromNullInt64(t *testing.T) {
	assert.Equal(t, 42, storage.NewIntegerFromNullInt64[int](sql.NullInt64{Int64: 42, Valid: true}))
	assert.Equal(t, 0, storage.NewIntegerFromNullInt64[int](sql.NullInt64{}))
	assert.Equal(t, int32(42), storage.NewIntegerFromNullInt64[int32](sql.NullInt64{Int64: 42, Valid: true}))
}
