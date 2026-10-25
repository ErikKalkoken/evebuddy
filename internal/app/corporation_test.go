package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestDivisionDefaultWalletName(t *testing.T) {
	xassert.Equal(t, "Master Wallet", app.Division1.DefaultWalletName())
	xassert.Equal(t, "", app.DivisionZero.DefaultWalletName())
}

func TestDivisionDefaultHangarName(t *testing.T) {
	xassert.Equal(t, "1st Division", app.Division1.DefaultHangarName())
	xassert.Equal(t, "7th Division", app.Division7.DefaultHangarName())
	xassert.Equal(t, "", app.DivisionZero.DefaultHangarName())
}

func TestCorporation_IDorZero(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var c *app.Corporation
		xassert.Equal(t, 0, c.IDOrZero())
	})
	t.Run("not nil", func(t *testing.T) {
		c := &app.Corporation{ID: 42}
		xassert.Equal(t, 42, c.IDOrZero())
	})
}

func TestCorporation_NameorZero(t *testing.T) {
	t.Run("corp is nil", func(t *testing.T) {
		var c *app.Corporation
		xassert.Equal(t, "", c.NameOrZero())
	})
	t.Run("corp is not nil", func(t *testing.T) {
		c := &app.Corporation{EveCorporation: &app.EveCorporation{Name: "Alpha"}}
		xassert.Equal(t, "Alpha", c.NameOrZero())
	})
	t.Run("eve corp is nil", func(t *testing.T) {
		c := &app.Corporation{}
		xassert.Equal(t, "", c.NameOrZero())
	})
}
