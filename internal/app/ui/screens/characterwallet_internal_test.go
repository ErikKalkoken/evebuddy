package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterWallet(t *testing.T) {
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("can render mobile=%v", isMobile), func(t *testing.T) {
			// given
			db, st, _ := testutil.NewDBOnDisk(t)
			defer db.Close()
			a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
				App:      test.NewTempApp(t),
				IsMobile: isMobile,
				Storage:  st,
			}))
			// when
			test.WidgetRenderer(a)
		})
	}
	t.Run("reports balance of current character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWallet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.OnBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1000.0),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.journal.character.Store(c)
		// when
		a.journal.updateBalanceCharacter(t.Context())
		// then
		xassert.Equal(t, []optional.Optional[float64]{optional.New(1000.0)}, reported)
	})
}
