package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCorporation_IDorZero(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		c := new(app.Corporation)
		xassert.Equal(t, 0, c.IDorZero())
	})
	t.Run("not nil", func(t *testing.T) {
		c := &app.Corporation{ID: 42}
		xassert.Equal(t, 42, c.IDorZero())
	})
}
