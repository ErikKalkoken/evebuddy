package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestESIStatusIsOK(t *testing.T) {
	t.Run("ok when no error message", func(t *testing.T) {
		s := app.ESIStatus{}
		xassert.Equal(t, true, s.IsOK())
	})
	t.Run("not ok when there is an error message", func(t *testing.T) {
		s := app.ESIStatus{ErrorMessage: "boom"}
		xassert.Equal(t, false, s.IsOK())
	})
}
