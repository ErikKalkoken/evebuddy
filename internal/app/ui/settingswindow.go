package ui

import (
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type settingsWindow struct {
	content fyne.CanvasObject
	u       *UI
	window  fyne.Window
}

func (u *UI) showSettingsWindow() {
	if u.settingsWindow != nil {
		u.settingsWindow.Show()
		return
	}
	sw, err := u.newSettingsWindow()
	if err != nil {
		panic(err)
	}
	w := u.fyneApp.NewWindow(u.makeWindowTitle("Settings"))
	w.SetContent(sw.content)
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.Show()
	w.SetCloseIntercept(func() {
		u.settingsWindow = nil
		w.Hide()
	})
	u.settingsWindow = w
	sw.window = w
}

func (u *UI) newSettingsWindow() (*settingsWindow, error) {
	sw := &settingsWindow{u: u}
	tabs := container.NewAppTabs(
		container.NewTabItem("General", sw.makeGeneralPage()),
		container.NewTabItem("Eve Online", sw.makeEVEOnlinePage()),
		container.NewTabItem("Notifications", sw.makeNotificationPage()),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	sw.content = tabs
	return sw, nil
}

// Themes
const (
	themeAuto  = "Auto"
	themeDark  = "Dark"
	themeLight = "Light"
)

func (w *settingsWindow) makeGeneralPage() fyne.CanvasObject {
	// theme
	themeRadio := widget.NewRadioGroup(
		[]string{themeAuto, themeDark, themeLight}, func(s string) {},
	)
	current := w.u.fyneApp.Preferences().StringWithFallback(settingTheme, settingThemeDefault)
	themeRadio.SetSelected(current)
	themeRadio.OnChanged = func(string) {
		w.u.themeSet(themeRadio.Selected)
		w.u.fyneApp.Preferences().SetString(settingTheme, themeRadio.Selected)
	}

	// system tray
	sysTrayCheck := kxwidget.NewToggle(func(b bool) {
		w.u.fyneApp.Preferences().SetBool(settingSysTrayEnabled, b)
	})
	sysTrayEnabled := w.u.fyneApp.Preferences().BoolWithFallback(
		settingSysTrayEnabled,
		settingSysTrayEnabledDefault,
	)
	sysTrayCheck.SetState(sysTrayEnabled)

	// cache
	clearBtn := widget.NewButton("Clear NOW", func() {
		m := kxmodal.NewProgressInfinite(
			"Clearing cache...",
			"",
			func() error {
				n, err := w.u.EveImageService.ClearCache()
				if err != nil {
					return err
				}
				slog.Info("Cleared image cache", "count", n)
				return nil
			},
			w.window,
		)
		m.OnSuccess = func() {
			d := dialog.NewInformation("Image cache", "Image cache cleared", w.window)
			d.Show()
		}
		m.OnError = func(err error) {
			slog.Error("Failed to clear image cache", "error", err)
			d := newErrorDialog("Failed to clear image cache", err, w.u.window)
			d.Show()
		}
		m.Start()
	})
	var cacheSize string
	s, err := w.u.EveImageService.Size()
	if err != nil {
		cacheSize = "?"
	} else {
		cacheSize = humanize.Bytes(uint64(s))
	}
	cacheHintText := fmt.Sprintf("Clear the local image cache (%s)", cacheSize)

	settings := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Style",
				Widget:   themeRadio,
				HintText: "Choose the style",
			},
			{
				Text:     "Close button",
				Widget:   sysTrayCheck,
				HintText: "App will minimize to system tray when closed (requires restart)",
			},
			{
				Text:     "Image cache",
				Widget:   container.NewHBox(clearBtn),
				HintText: cacheHintText,
			},
		}}
	reset := func() {
		themeRadio.SetSelected(settingThemeDefault)
		sysTrayCheck.SetState(settingSysTrayEnabledDefault)
	}
	return makePage("General settings", settings, reset)
}

func (w *settingsWindow) makeEVEOnlinePage() fyne.CanvasObject {
	// max mails
	maxMails := kxwidget.NewSliderWithValue(0, settingMaxMailsMax)
	v1 := w.u.fyneApp.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault)
	maxMails.SetValue(float64(v1))
	maxMails.OnChangeEnded = func(v float64) {
		w.u.fyneApp.Preferences().SetInt(settingMaxMails, int(v))
	}

	// max transactions
	maxTransactions := kxwidget.NewSliderWithValue(0, settingMaxWalletTransactionsMax)
	v2 := w.u.fyneApp.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
	maxTransactions.SetValue(float64(v2))
	maxTransactions.OnChangeEnded = func(v float64) {
		w.u.fyneApp.Preferences().SetInt(settingMaxWalletTransactions, int(v))
	}

	settings := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Maximum mails",
				Widget:   maxMails,
				HintText: "Maximum number of mails downloaded. 0 = unlimited.",
			},
			{
				Text:     "Maximum wallet transaction",
				Widget:   maxTransactions,
				HintText: "Maximum number of wallet transaction downloaded. 0 = unlimited.",
			},
		},
	}
	x := func() {
		maxMails.SetValue(settingMaxMailsDefault)
		maxTransactions.SetValue(settingMaxWalletTransactionsDefault)
	}
	return makePage("Eve Online settings", settings, x)
}

func (w *settingsWindow) makeNotificationPage() fyne.CanvasObject {
	s1 := widget.NewForm()

	// mail toogle
	mailEnabledCheck := kxwidget.NewToggle(func(b bool) {
		w.u.fyneApp.Preferences().SetBool(settingNotifyMailsEnabled, b)
	})
	mailEnabledCheck.On = w.u.fyneApp.Preferences().BoolWithFallback(
		settingNotifyMailsEnabled,
		settingNotifyMailsEnabledDefault,
	)
	s1.AppendItem(&widget.FormItem{
		Text:     "Mail",
		Widget:   mailEnabledCheck,
		HintText: "Wether to notify new mails",
	})

	// notifications toogle
	communicationsEnabledCheck := kxwidget.NewToggle(func(on bool) {
		w.u.fyneApp.Preferences().SetBool(settingNotifyCommunicationsEnabled, on)
	})
	communicationsEnabledCheck.On = w.u.fyneApp.Preferences().BoolWithFallback(
		settingNotifyCommunicationsEnabled,
		settingNotifyCommunicationsEnabledDefault,
	)
	s1.AppendItem(&widget.FormItem{
		Text:     "Communications",
		Widget:   communicationsEnabledCheck,
		HintText: "Wether to notify new communications",
	})

	// max age
	maxAge := kxwidget.NewSliderWithValue(1, settingMaxAgeMax)
	v := w.u.fyneApp.Preferences().IntWithFallback(settingMaxAge, settingMaxAgeDefault)
	maxAge.SetValue(float64(v))
	maxAge.OnChangeEnded = func(v float64) {
		w.u.fyneApp.Preferences().SetInt(settingMaxAge, int(v))
	}
	s1.AppendItem(&widget.FormItem{
		Text:     "Max age",
		Widget:   maxAge,
		HintText: "Max age in hours. Older mails and communications will not be notified.",
	})

	s2 := widget.NewForm()
	categoriesAndTypes := make(map[evenotification.Category][]string)
	for _, n := range evenotification.SupportedTypes() {
		c := evenotification.Type2category[n]
		categoriesAndTypes[c] = append(categoriesAndTypes[c], string(n))
	}
	categories := make([]evenotification.Category, 0)
	for c := range categoriesAndTypes {
		categories = append(categories, c)
	}
	slices.Sort(categories)
	typesEnabled := set.NewFromSlice(w.u.fyneApp.Preferences().StringList(settingNotificationsTypesEnabled))
	groups := make([]*widget.CheckGroup, 0)
	for _, c := range categories {
		nts := categoriesAndTypes[c]
		selected := make([]string, 0)
		for _, nt := range nts {
			if typesEnabled.Contains(nt) {
				selected = append(selected, nt)
			}
		}
		cg := widget.NewCheckGroup(nts, nil)
		cg.Selected = selected
		cg.OnChanged = func(s []string) {
			enabled := make([]string, 0)
			for _, cg := range groups {
				enabled = slices.Concat(enabled, cg.Selected)
			}
			w.u.fyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, enabled)
		}
		s2.AppendItem(widget.NewFormItem(c.String(), cg))
		enableAll := widget.NewButton("Enable all", func() {
			cg.SetSelected(cg.Options)
		})
		disableAll := widget.NewButton("Disable all", func() {
			cg.SetSelected([]string{})
		})
		s2.Append("", container.NewHBox(enableAll, disableAll))
		s2.Append("", container.NewPadded())
		groups = append(groups, cg)
	}
	title1 := widget.NewLabel("Global")
	title1.TextStyle.Bold = true
	title2 := widget.NewLabel("Communication Types")
	title2.TextStyle.Bold = true
	c := container.NewVBox(
		title1,
		s1,
		container.NewPadded(),
		title2,
		s2,
	)
	reset := func() {
		mailEnabledCheck.SetState(settingNotifyMailsEnabledDefault)
		communicationsEnabledCheck.SetState(settingNotifyCommunicationsEnabledDefault)
		maxAge.SetValue(settingMaxAgeDefault)
		for _, cg := range groups {
			cg.SetSelected([]string{})
		}
	}
	return makePage("Notification settings", c, reset)
}

func makePage(title string, content fyne.CanvasObject, resetSettings func()) fyne.CanvasObject {
	l := widget.NewLabel(title)
	l.Importance = widget.HighImportance
	l.TextStyle.Bold = true
	return container.NewBorder(
		container.NewVBox(l, widget.NewSeparator()),
		container.NewHBox(widget.NewButton("Reset", resetSettings)),
		nil,
		nil,
		container.NewVScroll(content),
	)
}
