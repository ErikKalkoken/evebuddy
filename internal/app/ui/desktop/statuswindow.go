package desktop

import (
	"context"

	"fyne.io/fyne/v2"
)

func (u *DesktopUI) showStatusWindow() {
	if u.statusWindow != nil {
		u.statusWindow.Show()
		return
	}
	w := u.FyneApp.NewWindow(u.makeWindowTitle("Status"))
	a := u.NewStatusArea()
	a.Refresh()
	w.SetContent(a.Content)
	w.Resize(fyne.Size{Width: 1100, Height: 500})
	ctx, cancel := context.WithCancel(context.Background())
	a.StartTicker(ctx)
	w.SetOnClosed(func() {
		cancel()
		u.statusWindow = nil
	})
	u.statusWindow = w
	w.Show()
}
