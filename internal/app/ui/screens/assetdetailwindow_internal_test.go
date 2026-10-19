package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestAssetLocationPathDisplay(t *testing.T) {
	t.Run("joins path elements", func(t *testing.T) {
		// given
		r := assetRow{locationPath: []string{"Jita IV - Moon 4", "Item Hangar", "Container"}}
		// when
		got := assetLocationPathDisplay(r)
		// then
		xassert.Equal(t, "Jita IV - Moon 4 / Item Hangar / Container", got)
	})
	t.Run("shows dash when path is empty", func(t *testing.T) {
		// given
		r := assetRow{}
		// when
		got := assetLocationPathDisplay(r)
		// then
		xassert.Equal(t, "-", got)
	})
}

func TestShowAssetDetails(t *testing.T) {
	t.Run("can show asset with location", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		owner := factory.CreateEveEntityCharacter()
		el := factory.CreateEveLocationStation()
		r := assetRow{
			itemID:       42,
			location:     el.ToEveLocationShort(),
			locationPath: []string{"Item Hangar"},
			owner:        owner,
			price:        optional.New(1000.0),
			total:        optional.New(5000.0),
			typeName:     "Tritanium",
		}
		// when
		ShowAssetDetails(u, r)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("asset-%d-%d", owner.ID, r.itemID))
		assert.False(t, created)
	})
	t.Run("can show asset without location for a corporation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		owner := factory.CreateEveEntityCorporation()
		r := assetRow{
			itemID:  42,
			owner:   owner,
			variant: app.VariantBPC,
		}
		// when
		ShowAssetDetails(u, r)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("asset-%d-%d", owner.ID, r.itemID))
		assert.False(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		owner := factory.CreateEveEntityCharacter()
		r := assetRow{itemID: 42, owner: owner}
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("asset-%d-%d", owner.ID, r.itemID))
		// when
		ShowAssetDetails(u, r)
		// then
		assert.Nil(t, w.Content())
	})
}
