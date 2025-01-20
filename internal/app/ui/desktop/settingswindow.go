package desktop

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

type SettingsArea struct {
	content fyne.CanvasObject
	u       *DesktopUI
	window  fyne.Window
}

func (u *DesktopUI) showSettingsWindow() {
	if u.settingsWindow != nil {
		u.settingsWindow.Show()
		return
	}
	w := u.FyneApp.NewWindow(u.makeWindowTitle("Settings"))
	sw := u.NewSettingsArea()
	w.SetContent(sw.content)
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		u.settingsWindow = nil
	})
	u.settingsWindow = w
	sw.window = w
	w.Show()
}

func (u *DesktopUI) NewSettingsArea() *SettingsArea {
	sw := &SettingsArea{u: u}
	tabs := container.NewAppTabs(
		container.NewTabItem("General", func() fyne.CanvasObject {
			c, f := u.MakeGeneralSettingsPage(u.settingsWindow)
			return makePage("Desktop", c, f)
		}()),
		container.NewTabItem("Desktop", func() fyne.CanvasObject {
			c, f := u.MakeDesktopSettingsPage()
			return makePage("Desktop", c, f)
		}()),
		container.NewTabItem("Eve Online", func() fyne.CanvasObject {
			c, f := u.MakeEVEOnlinePage()
			return makePage("Desktop", c, f)
		}()),
		container.NewTabItem("Notifications", func() fyne.CanvasObject {
			c, f := u.MakeNotificationPage(u.settingsWindow)
			return makePage("Notifications", c, f)
		}()),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	sw.content = tabs
	return sw
}

func (u *DesktopUI) MakeDesktopSettingsPage() (fyne.CanvasObject, func()) {
	// // system tray
	sysTrayCheck := kxwidget.NewSwitch(func(b bool) {
		u.FyneApp.Preferences().SetBool(settingSysTrayEnabled, b)
	})
	sysTrayEnabled := u.FyneApp.Preferences().BoolWithFallback(
		settingSysTrayEnabled,
		settingSysTrayEnabledDefault,
	)
	sysTrayCheck.SetState(sysTrayEnabled)

	settings := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Close button",
				Widget:   sysTrayCheck,
				HintText: "App will minimize to system tray when closed (requires restart)",
			},
		}}
	reset := func() {
		sysTrayCheck.SetState(settingSysTrayEnabledDefault)
	}
	return settings, reset
}

func makePage(title string, content fyne.CanvasObject, resetSettings func()) fyne.CanvasObject {
	x := widget.NewLabel(title)
	x.TextStyle.Bold = true
	return container.NewBorder(
		container.NewVBox(x, widget.NewSeparator()),
		container.NewHBox(widget.NewButton("Reset", resetSettings)),
		nil,
		nil,
		container.NewScroll(content),
	)
}
