package desktop

import (
	"fyne.io/fyne/v2"
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
		u.RefreshCrossPages() //FIXME: Is this really needed?
	})
	w.Resize(fyne.Size{Width: 500, Height: 500})
	w.SetContent(u.AccountArea.Content)
	w.Show()
	u.AccountArea.OnSelectCharacter = func() {
		w.Hide()
	}
}
