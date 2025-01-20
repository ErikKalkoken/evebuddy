package desktop

import (
	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

type accountWindow struct {
	content fyne.CanvasObject
	u       *DesktopUI
}

func (u *DesktopUI) showAccountWindow() {
	if u.accountWindow != nil {
		u.accountWindow.Show()
		return
	}
	w := u.FyneApp.NewWindow(u.makeWindowTitle("Characters"))
	u.accountWindow = w
	w.SetOnClosed(func() {
		u.accountWindow = nil
		u.refreshCrossPages()
	})
	w.Resize(fyne.Size{Width: 500, Height: 500})
	w.SetContent(u.AccountArea.Content)
	w.Show()
	if err := u.AccountArea.Refresh(); err != nil {
		w.Hide()
		d := ui.NewErrorDialog("Failed to show characters", err, u.Window)
		d.Show()
	}
	u.AccountArea.OnSelectCharacter = func() {
		w.Hide()
	}
}
