package storage

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestConvertNoRowsError(t *testing.T) {
	t.Run("converts no rows error", func(t *testing.T) {
		got := convertGetError(sql.ErrNoRows)
		xassert.Equal(t, app.ErrNotFound, got)
	})
	t.Run("passes through other errors", func(t *testing.T) {
		err := errors.New("random error")
		got := convertGetError(err)
		xassert.Equal(t, err, got)
	})
}
