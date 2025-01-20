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
		container.NewTabItem("General", makePage(u.MakeGeneralSettingsPage(u.settingsWindow))),
		container.NewTabItem("Desktop", makePage(u.MakeDesktopSettingsPage())),
		container.NewTabItem("Eve Online", makePage(u.MakeEVEOnlinePage())),
		container.NewTabItem("Notifications", makePage(u.MakeNotificationPage(u.settingsWindow))),
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

func makePage(content fyne.CanvasObject, resetSettings func()) fyne.CanvasObject {
	return container.NewBorder(
		nil,
		container.NewHBox(widget.NewButton("Reset", resetSettings)),
		nil,
		nil,
		container.NewVScroll(content),
	)
}
