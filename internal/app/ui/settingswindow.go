package ui

import (
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/dustin/go-humanize"
)

// Settings
const (
	settingLastCharacterID                    = "settingLastCharacterID"
	settingMaxAge                             = "settingMaxAgeHours"
	settingMaxAgeDefault                      = 6  // hours
	settingMaxAgeMax                          = 24 // hours
	settingMaxMails                           = "settingMaxMails"
	settingMaxMailsDefault                    = 1_000
	settingMaxMailsMax                        = 10_000
	settingMaxWalletTransactions              = "settingMaxWalletTransactions"
	settingMaxWalletTransactionsDefault       = 1_000
	settingMaxWalletTransactionsMax           = 10_000
	settingNotifyCommunicationsEnabled        = "settingNotifyCommunicationsEnabled"
	settingNotifyCommunicationsEnabledDefault = false
	settingNotifyMailsEnabled                 = "settingNotifyMailsEnabled"
	settingNotifyMailsEnabledDefault          = false
	settingNotificationsTypesEnabled          = "settingNotificationsTypesEnabled"
	settingSysTrayEnabled                     = "settingSysTrayEnabled"
	settingSysTrayEnabledDefault              = false
	settingTheme                              = "settingTheme"
	settingThemeDefault                       = themeAuto
)

// Themes
const (
	themeAuto  = "Auto"
	themeDark  = "Dark"
	themeLight = "Light"
)

type settingsWindow struct {
	content fyne.CanvasObject
	ui      *ui
	window  fyne.Window
}

func (u *ui) showSettingsWindow() {
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

func (u *ui) newSettingsWindow() (*settingsWindow, error) {
	sw := &settingsWindow{ui: u}
	tabs := container.NewAppTabs(
		container.NewTabItem("General", sw.makeGeneralPage()),
		container.NewTabItem("Eve Online", sw.makeEVEOnlinePage()),
		container.NewTabItem("Notifications", sw.makeNotificationPage()),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	sw.content = tabs
	return sw, nil
}

func (w *settingsWindow) makeGeneralPage() fyne.CanvasObject {
	// theme
	themeRadio := widget.NewRadioGroup(
		[]string{themeAuto, themeDark, themeLight}, func(s string) {},
	)
	current := w.ui.fyneApp.Preferences().StringWithFallback(settingTheme, settingThemeDefault)
	themeRadio.SetSelected(current)
	themeRadio.OnChanged = func(string) {
		w.ui.themeSet(themeRadio.Selected)
		w.ui.fyneApp.Preferences().SetString(settingTheme, themeRadio.Selected)
	}

	// system tray
	sysTrayEnabled := w.ui.fyneApp.Preferences().BoolWithFallback(settingSysTrayEnabled, settingSysTrayEnabledDefault)
	sysTrayCheck := widget.NewCheck("Minimize to tray", nil)
	sysTrayCheck.SetChecked(sysTrayEnabled)
	sysTrayCheck.OnChanged = func(b bool) {
		w.ui.fyneApp.Preferences().SetBool(settingSysTrayEnabled, sysTrayCheck.Checked)
	}

	// cache
	clearBtn := widget.NewButton("Clear NOW", func() {
		d := dialog.NewConfirm(
			"Clear image cache",
			"Are you sure you want to clear the image cache?",
			func(confirmed bool) {
				if !confirmed {
					return
				}
				count, err := w.ui.EveImageService.ClearCache()
				if err != nil {
					slog.Error(err.Error())
					w.ui.showErrorDialog("Failed to clear image cache", err)
				} else {
					slog.Info("Cleared images cache", "count", count)
				}
			},
			w.window,
		)
		d.Show()
	})
	var cacheSize string
	s, err := w.ui.EveImageService.Size()
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
		sysTrayCheck.SetChecked(settingSysTrayEnabledDefault)
	}
	return makePage("General settings", settings, reset)
}

func (w *settingsWindow) makeEVEOnlinePage() fyne.CanvasObject {
	// max mails
	mm := w.ui.fyneApp.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault)
	maxMails := widgets.NewSlider(0, settingMaxMailsMax, mm)
	maxMails.OnChangeEnded = func(v int) {
		w.ui.fyneApp.Preferences().SetInt(settingMaxMails, v)
	}

	// max transactions
	mwt := w.ui.fyneApp.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
	maxTransactions := widgets.NewSlider(0, settingMaxWalletTransactionsMax, mwt)
	maxTransactions.OnChangeEnded = func(v int) {
		w.ui.fyneApp.Preferences().SetInt(settingMaxWalletTransactions, v)
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
	mailEnabledCheck := widget.NewCheck("Notif new mails", nil)
	mailEnabledCheck.Checked = w.ui.fyneApp.Preferences().BoolWithFallback(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault)
	mailEnabledCheck.OnChanged = func(b bool) {
		w.ui.fyneApp.Preferences().SetBool(settingNotifyMailsEnabled, mailEnabledCheck.Checked)
	}
	s1.AppendItem(&widget.FormItem{
		Text:     "Mail",
		Widget:   mailEnabledCheck,
		HintText: "Wether to notify new mails",
	})

	// notifications toogle
	communicationsEnabledCheck := widget.NewCheck("Notify new communications", nil)
	communicationsEnabledCheck.Checked = w.ui.fyneApp.Preferences().BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault)
	communicationsEnabledCheck.OnChanged = func(b bool) {
		w.ui.fyneApp.Preferences().SetBool(settingNotifyCommunicationsEnabled, communicationsEnabledCheck.Checked)
	}
	s1.AppendItem(&widget.FormItem{
		Text:     "Communications",
		Widget:   communicationsEnabledCheck,
		HintText: "Wether to notify new communications",
	})

	// max age
	ma := w.ui.fyneApp.Preferences().IntWithFallback(settingMaxAge, settingMaxAgeDefault)
	maxAge := widgets.NewSlider(1, settingMaxAgeMax, ma)
	maxAge.OnChangeEnded = func(v int) {
		w.ui.fyneApp.Preferences().SetInt(settingMaxAge, v)
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
	typesEnabled := set.NewFromSlice(w.ui.fyneApp.Preferences().StringList(settingNotificationsTypesEnabled))
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
			w.ui.fyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, enabled)
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
		mailEnabledCheck.SetChecked(settingNotifyMailsEnabledDefault)
		communicationsEnabledCheck.SetChecked(settingNotifyCommunicationsEnabledDefault)
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
