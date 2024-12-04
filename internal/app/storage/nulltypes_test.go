package storage_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/stretchr/testify/assert"
)

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
