package ui

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestInfoWindow_CanRenderLocationInfo(t *testing.T) {
	test.ApplyTheme(t, test.Theme())
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	ctx := context.Background()
	t.Run("can render full location", func(t *testing.T) {
		l := factory.CreateEveLocationStation()
		iw := newInfoWindow(u)
		a := newLocationInfo(iw, l.ID)
		a.update(ctx)
		test.RenderObjectToMarkup(a)
	})
	t.Run("can render minimal location", func(t *testing.T) {
		l := factory.CreateEveLocationEmptyStructure()
		iw := newInfoWindow(u)
		a := newLocationInfo(iw, l.ID)
		a.update(ctx)
		test.RenderObjectToMarkup(a)
	})
}
