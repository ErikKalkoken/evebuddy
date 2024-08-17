package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/pkg/set"
	"github.com/dustin/go-humanize"
)

// Setting keys
const (
	settingLastCharacterID              = "settingLastCharacterID"
	settingMaxMails                     = "settingMaxMails"
	settingMaxMailsDefault              = 1_000
	settingMaxWalletTransactions        = "settingMaxWalletTransactions"
	settingMaxWalletTransactionsDefault = 10_000
	settingTheme                        = "settingTheme"
	settingThemeDefault                 = themeAuto
	settingSysTrayEnabled               = "settingSysTrayEnabled"
	settingNotificationsEnabled         = "settingNotificationsEnabled"
	settingNotificationsTypesEnabled    = "settingNotificationsTypesEnabled"
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
	w.Resize(fyne.Size{Width: 1100, Height: 500})
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

	themeRadio := widget.NewRadioGroup(
		[]string{themeAuto, themeDark, themeLight}, func(s string) {},
	)
	name := w.ui.fyneApp.Preferences().StringWithFallback(settingTheme, settingThemeDefault)
	themeRadio.SetSelected(name)

	var cacheSize string
	s, err := w.ui.EveImageService.Size()
	if err != nil {
		cacheSize = "?"
	} else {
		cacheSize = humanize.Bytes(uint64(s))
	}
	cacheHintText := fmt.Sprintf("Clear the local image cache (%s)", cacheSize)

	sysTrayEnabled := w.ui.fyneApp.Preferences().BoolWithFallback(settingSysTrayEnabled, true)
	sysTrayCheck := widget.NewCheck("Show in system tray", nil)
	sysTrayCheck.SetChecked(sysTrayEnabled)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Theme",
				Widget:   themeRadio,
				HintText: "Chose the preferred theme",
			},
			{
				Text:     "System Tray",
				Widget:   sysTrayCheck,
				HintText: "Show icon in system tray (requires restart)",
			},
			{
				Text:     "Image cache",
				Widget:   container.NewHBox(clearBtn),
				HintText: cacheHintText,
			},
		},
		OnSubmit: func() {
			w.ui.themeSet(themeRadio.Selected)
			w.ui.fyneApp.Preferences().SetString(settingTheme, themeRadio.Selected)
			w.ui.fyneApp.Preferences().SetBool(settingSysTrayEnabled, sysTrayCheck.Checked)
		},
		OnCancel: func() {
			w.window.Hide()
		},
	}
	return makePage("General settings", form)
}

func (w *settingsWindow) makeEVEOnlinePage() fyne.CanvasObject {
	maxMails := w.ui.fyneApp.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault)
	maxMailsEntry := widget.NewEntry()
	maxMailsEntry.SetText(strconv.Itoa(maxMails))
	maxMailsEntry.Validator = newPositiveNumberValidator()

	maxTransactions := w.ui.fyneApp.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
	maxTransactionsEntry := widget.NewEntry()
	maxTransactionsEntry.SetText(strconv.Itoa(maxTransactions))
	maxTransactionsEntry.Validator = newPositiveNumberValidator()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Maximum mails",
				Widget:   maxMailsEntry,
				HintText: "Maximum number of mails downloaded. 0 = unlimited.",
			},
			{
				Text:     "Maximum wallet transaction",
				Widget:   maxTransactionsEntry,
				HintText: "Maximum number of wallet transaction downloaded. 0 = unlimited.",
			},
		},
		OnSubmit: func() {
			maxMails, err := strconv.Atoi(maxMailsEntry.Text)
			if err != nil {
				return
			}
			w.ui.fyneApp.Preferences().SetInt(settingMaxMails, maxMails)
			maxTransactions, err := strconv.Atoi(maxTransactionsEntry.Text)
			if err != nil {
				return
			}
			w.ui.fyneApp.Preferences().SetInt(settingMaxWalletTransactions, maxTransactions)
		},
		OnCancel: func() {
			w.window.Hide()
		},
	}
	return makePage("Eve Online settings", form)
}

func (w *settingsWindow) makeNotificationPage() fyne.CanvasObject {
	items := make([]*widget.FormItem, 0)

	notificationsEnabledCheck := widget.NewCheck("Notifications enabled", nil)
	notificationsEnabledCheck.Checked = w.ui.fyneApp.Preferences().Bool(settingNotificationsEnabled)
	items = append(items, &widget.FormItem{
		Text:     "Master",
		Widget:   notificationsEnabledCheck,
		HintText: "Wether notifications are enabled",
	})

	categories := make(map[evenotification.Category][]string)
	for _, n := range evenotification.SupportedTypes() {
		c := evenotification.Type2category[n]
		categories[c] = append(categories[c], string(n))
	}
	typesEnabled := set.NewFromSlice(w.ui.fyneApp.Preferences().StringList(settingNotificationsTypesEnabled))
	groups := make([]*widget.CheckGroup, 0)
	for c, nts := range categories {
		selected := make([]string, 0)
		for _, nt := range nts {
			if typesEnabled.Has(nt) {
				selected = append(selected, nt)
			}
		}
		cg := widget.NewCheckGroup(nts, nil)
		cg.Selected = selected
		items = append(items, widget.NewFormItem(c.String(), cg))
		groups = append(groups, cg)
	}

	form := &widget.Form{
		Items: items,
		OnSubmit: func() {
			w.ui.fyneApp.Preferences().SetBool(settingNotificationsEnabled, notificationsEnabledCheck.Checked)
			enabled := make([]string, 0)
			for _, cg := range groups {
				enabled = slices.Concat(enabled, cg.Selected)
			}
			w.ui.fyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, enabled)
		},
		OnCancel: func() {
			w.window.Hide()
		},
	}
	return makePage("Eve Online settings", form)
}

// newPositiveNumberValidator ensures entry is a positive number (incl. zero).
func newPositiveNumberValidator() fyne.StringValidator {
	myErr := errors.New("must be positive number")
	return func(text string) error {
		val, err := strconv.Atoi(text)
		if err != nil {
			return myErr
		}
		if val < 0 {
			return myErr
		}
		return nil
	}
}

func makePage(title string, content fyne.CanvasObject) fyne.CanvasObject {
	l := widget.NewLabel(title)
	l.Importance = widget.HighImportance
	l.TextStyle.Bold = true
	return container.NewBorder(
		container.NewVBox(l, widget.NewSeparator()),
		nil,
		nil,
		nil,
		container.NewVScroll(content),
	)
}
