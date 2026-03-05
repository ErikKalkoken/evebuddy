package infowindow

// import (
// 	"context"
// 	"testing"

// 	"fyne.io/fyne/v2/test"

// 	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
// )

// func TestInfoWindow_CanRenderLocationInfo(t *testing.T) {
// 	test.ApplyTheme(t, test.Theme())
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()
// 	u := MakeFakeBaseUI(st, test.NewTempApp(t), true)
// 	ctx := context.Background()
// 	makeInfoWindow := func() *InfoWindow {
// 		return NewInfoWindow(InfoWindowParams{
// 			cs:       u.cs,
// 			eis:      u.eis,
// 			eus:      u.eus,
// 			isMobile: u.isMobile,
// 			js:       u.js,
// 			settings: u.settings,
// 			scs:      u.scs,
// 			u:        u,
// 			w:        u.MainWindow(),
// 		})
// 	}
// 	t.Run("can render full location", func(t *testing.T) {
// 		l := factory.CreateEveLocationStation()
// 		iw := makeInfoWindow()
// 		a := newLocationInfo(iw, l.ID)
// 		a.update(ctx)
// 		test.RenderObjectToMarkup(a)
// 	})
// 	t.Run("can render minimal location", func(t *testing.T) {
// 		l := factory.CreateEveLocationEmptyStructure()
// 		iw := makeInfoWindow()
// 		a := newLocationInfo(iw, l.ID)
// 		a.update(ctx)
// 		test.RenderObjectToMarkup(a)
// 	})
// }
