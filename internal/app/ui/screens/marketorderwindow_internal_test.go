package screens

import (
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestMarketOrderRow_VariantDisplay(t *testing.T) {
	t.Run("shows buy for buy orders", func(t *testing.T) {
		// given
		r := marketOrderRow{IsBuyOrder: optional.New(true)}
		// when
		got := r.variantDisplay()
		// then
		xassert.Equal(t, "buy", got)
	})
	t.Run("shows sell for sell orders", func(t *testing.T) {
		// given
		r := marketOrderRow{IsBuyOrder: optional.New(false)}
		// when
		got := r.variantDisplay()
		// then
		xassert.Equal(t, "sell", got)
	})
	t.Run("shows sell when order variant is unknown", func(t *testing.T) {
		// given
		r := marketOrderRow{}
		// when
		got := r.variantDisplay()
		// then
		xassert.Equal(t, "sell", got)
	})
}

func TestMarketOrderRow_ExpiresDisplay(t *testing.T) {
	t.Run("shows expiry date of running order", func(t *testing.T) {
		// given
		expires := time.Now().Add(24 * time.Hour)
		r := marketOrderRow{expires: expires}
		// when
		got := r.expiresDisplay()
		// then
		xassert.Equal(t, expires.Format(app.DateTimeFormat), got)
	})
	t.Run("shows dash for expired order", func(t *testing.T) {
		// given
		r := marketOrderRow{expires: time.Now().Add(-time.Hour)}
		// when
		got := r.expiresDisplay()
		// then
		xassert.Equal(t, "-", got)
	})
}

func TestMakeBoolLabel(t *testing.T) {
	t.Run("shows yes for true", func(t *testing.T) {
		// when
		got := makeBoolLabel(true)
		// then
		xassert.Equal(t, "Yes", got.Text)
		xassert.Equal(t, widget.SuccessImportance, got.Importance)
	})
	t.Run("shows no for false", func(t *testing.T) {
		// when
		got := makeBoolLabel(false)
		// then
		xassert.Equal(t, "No", got.Text)
		xassert.Equal(t, widget.DangerImportance, got.Importance)
	})
}

func TestShowMarketOrderWindow(t *testing.T) {
	t.Run("can show buy order", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		owner := factory.CreateEveEntityCharacter()
		r := marketOrderRow{
			characterID:   owner.ID,
			characterName: owner.Name,
			escrow:        optional.New(1000.0),
			expires:       time.Now().Add(24 * time.Hour),
			IsBuyOrder:    optional.New(true),
			issued:        time.Now(),
			location:      factory.CreateEveLocationStation().ToEveLocationShort(),
			minVolume:     optional.New[int64](1),
			orderID:       42,
			owner:         owner,
			state:         app.OrderOpen,
		}
		// when
		ShowMarketOrderWindow(u, r)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("market-order-%d-%d", owner.ID, r.orderID))
		assert.False(t, created)
	})
	t.Run("can show sell order", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		owner := factory.CreateEveEntityCharacter()
		r := marketOrderRow{
			characterID:   owner.ID,
			characterName: owner.Name,
			expires:       time.Now().Add(-time.Hour),
			IsBuyOrder:    optional.New(false),
			issued:        time.Now(),
			location:      factory.CreateEveLocationStation().ToEveLocationShort(),
			orderID:       42,
			owner:         owner,
			state:         app.OrderOpen,
		}
		// when
		ShowMarketOrderWindow(u, r)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("market-order-%d-%d", owner.ID, r.orderID))
		assert.False(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		r := marketOrderRow{characterID: 7, orderID: 42}
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("market-order-%d-%d", r.characterID, r.orderID))
		// when
		ShowMarketOrderWindow(u, r)
		// then
		assert.Nil(t, w.Content())
	})
}
