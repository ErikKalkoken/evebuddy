package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCorporationWalletLabel(t *testing.T) {
	t.Run("shows name with division", func(t *testing.T) {
		// when
		got := corporationWalletLabel(app.Division3, "Ops")
		// then
		xassert.Equal(t, "[3] Ops", got)
	})
	t.Run("shows default name when name is empty", func(t *testing.T) {
		// when
		got := corporationWalletLabel(app.Division1, "")
		// then
		xassert.Equal(t, "[1] Master Wallet", got)
	})
}

func TestCorporationWallets(t *testing.T) {
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("can render mobile=%v", isMobile), func(t *testing.T) {
			// given
			db, st, _ := testutil.NewDBOnDisk(t)
			defer db.Close()
			a := NewCorporationWallets(testdouble.NewUIFake(testdouble.UIParams{
				App:      test.NewTempApp(t),
				IsMobile: isMobile,
				Storage:  st,
			}))
			// when
			test.WidgetRenderer(a)
		})
	}
	t.Run("reports total balance of current corporation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWallets(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.OnBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		c := createCorporationWithWalletBalances(factory)
		a.divisions.corporation.Store(c)
		// when
		a.divisions.Update(t.Context())
		// then
		xassert.Equal(t, []optional.Optional[float64]{optional.New(1234.5)}, reported)
	})
}
