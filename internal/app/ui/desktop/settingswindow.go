package desktop

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
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
	w := u.FyneApp.NewWindow(u.MakeWindowTitle("Settings"))
	sw := u.NewSettingsArea(w)
	w.SetContent(sw.content)
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		u.settingsWindow = nil
	})
	sw.window = w
	w.Show()
}

func (u *DesktopUI) NewSettingsArea(w fyne.Window) *SettingsArea {
	sw := &SettingsArea{u: u, window: w}
	tabs := container.NewAppTabs(
		container.NewTabItem("General", func() fyne.CanvasObject {
			c, items := u.MakeGeneralSettingsPage(w)
			return makePage("Desktop", c, items)
		}()),
		container.NewTabItem("Desktop", func() fyne.CanvasObject {
			c, items := u.MakeDesktopSettingsPage()
			return makePage("Desktop", c, items)
		}()),
		container.NewTabItem("Eve Online", func() fyne.CanvasObject {
			c, f := u.MakeEVEOnlinePage()
			return makePage("Desktop", c, f)
		}()),
		container.NewTabItem("Notifications", func() fyne.CanvasObject {
			c, f := u.MakeNotificationGeneralPage(w)
			return makePage("Notifications", c, f)
		}()),
		container.NewTabItem("Communications", func() fyne.CanvasObject {
			c, f := u.MakeNotificationTypesPage(w)
			return makePage("Communications", c, f)
		}()),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	sw.content = tabs
	return sw
}

func (u *DesktopUI) MakeDesktopSettingsPage() (fyne.CanvasObject, []*fyne.MenuItem) {
	// system tray
	sysTrayCheck := kxwidget.NewSwitch(func(b bool) {
		u.FyneApp.Preferences().SetBool(settingSysTrayEnabled, b)
	})
	sysTrayEnabled := u.FyneApp.Preferences().BoolWithFallback(
		settingSysTrayEnabled,
		settingSysTrayEnabledDefault,
	)
	sysTrayCheck.SetState(sysTrayEnabled)

	// window
	resetWindow := widget.NewButton("Reset main window size", func() {
		u.Window.Resize(fyne.NewSize(settingWindowWidthDefault, settingWindowHeightDefault))
	})

	settings := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Close button",
				Widget:   sysTrayCheck,
				HintText: "App will minimize to system tray when closed (requires restart)",
			},
			{
				Text:     "Window",
				Widget:   resetWindow,
				HintText: "Resets window size to defaults",
			},
		}}
	reset := &fyne.MenuItem{
		Label: "Reset",
		Action: func() {
			sysTrayCheck.SetState(settingSysTrayEnabledDefault)
		},
	}
	return settings, []*fyne.MenuItem{reset}
}

func makePage(title string, content fyne.CanvasObject, items []*fyne.MenuItem) fyne.CanvasObject {
	t := widget.NewLabel(title)
	t.TextStyle.Bold = true
	top := container.NewHBox(t, layout.NewSpacer(), container.NewHBox(widgets.NewIconButtonWithMenu(
		theme.MenuIcon(), fyne.NewMenu("", items...),
	)))
	return container.NewBorder(
		container.NewVBox(top, widget.NewSeparator()),
		nil,
		nil,
		nil,
		container.NewScroll(content),
	)
}
