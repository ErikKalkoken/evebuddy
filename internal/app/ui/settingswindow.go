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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
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
	submit := widget.NewButtonWithIcon("Apply", theme.ConfirmIcon(), nil)
	submit.Importance = widget.HighImportance
	submit.Disable()

	form := widget.NewForm()

	themeRadio := widget.NewRadioGroup(
		[]string{themeAuto, themeDark, themeLight}, func(s string) {},
	)
	current := w.ui.fyneApp.Preferences().StringWithFallback(settingTheme, settingThemeDefault)
	themeRadio.SetSelected(current)
	form.AppendItem(&widget.FormItem{
		Text:     "Theme",
		Widget:   themeRadio,
		HintText: "Chose the preferred theme",
	})

	sysTrayEnabled := w.ui.fyneApp.Preferences().BoolWithFallback(settingSysTrayEnabled, true)
	sysTrayCheck := widget.NewCheck("Show in system tray", nil)
	sysTrayCheck.SetChecked(sysTrayEnabled)
	form.AppendItem(&widget.FormItem{
		Text:     "System Tray",
		Widget:   sysTrayCheck,
		HintText: "Show icon in system tray (requires restart)",
	})

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
	form.AppendItem(&widget.FormItem{
		Text:     "Image cache",
		Widget:   container.NewHBox(clearBtn),
		HintText: cacheHintText,
	})

	content := container.NewVBox()
	content.Add(form)
	submit.OnTapped = func() {
		w.ui.themeSet(themeRadio.Selected)
		w.ui.fyneApp.Preferences().SetString(settingTheme, themeRadio.Selected)
		w.ui.fyneApp.Preferences().SetBool(settingSysTrayEnabled, sysTrayCheck.Checked)
	}
	cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		w.window.Hide()
	})
	content.Add(container.NewHBox(layout.NewSpacer(), cancel, submit))
	return makePage("General settings", content)
}

func (w *settingsWindow) makeEVEOnlinePage() fyne.CanvasObject {
	submit := widget.NewButtonWithIcon("Apply", theme.ConfirmIcon(), nil)
	submit.Importance = widget.HighImportance
	submit.Disable()

	form := widget.NewForm()
	mm := w.ui.fyneApp.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault)
	maxMails := widget.NewEntry()
	maxMails.SetText(strconv.Itoa(mm))
	maxMails.Validator = newPositiveNumberValidator()
	maxMails.OnChanged = func(s string) { submit.Enable() }
	form.AppendItem(&widget.FormItem{
		Text:     "Maximum mails",
		Widget:   maxMails,
		HintText: "Maximum number of mails downloaded. 0 = unlimited.",
	})

	mwt := w.ui.fyneApp.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
	maxTransactions := widget.NewEntry()
	maxTransactions.SetText(strconv.Itoa(mwt))
	maxTransactions.Validator = newPositiveNumberValidator()
	maxTransactions.OnChanged = func(s string) {
		submit.Enable()
	}
	form.AppendItem(&widget.FormItem{
		Text:     "Maximum wallet transaction",
		Widget:   maxTransactions,
		HintText: "Maximum number of wallet transaction downloaded. 0 = unlimited.",
	})

	content := container.NewVBox()
	content.Add(form)
	content.Add(container.NewPadded())

	submit.OnTapped = func() {
		if err := form.Validate(); err != nil {
			d := dialog.NewInformation("Invalid input", err.Error(), w.window)
			d.Show()
			return
		}
		mm, err := strconv.Atoi(maxMails.Text)
		if err != nil {
			return
		}
		w.ui.fyneApp.Preferences().SetInt(settingMaxMails, mm)
		mwt, err := strconv.Atoi(maxTransactions.Text)
		if err != nil {
			return
		}
		w.ui.fyneApp.Preferences().SetInt(settingMaxWalletTransactions, mwt)
		submit.Disable()
	}
	cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		w.window.Hide()
	})
	content.Add(container.NewHBox(layout.NewSpacer(), cancel, submit))
	return makePage("Eve Online settings", content)
}

func (w *settingsWindow) makeNotificationPage() fyne.CanvasObject {
	submit := widget.NewButtonWithIcon("Apply", theme.ConfirmIcon(), nil)
	submit.Importance = widget.HighImportance
	submit.Disable()

	content := container.NewVBox()
	form := widget.NewForm()

	notificationsEnabledCheck := widget.NewCheck("Notifications enabled", nil)
	notificationsEnabledCheck.Checked = w.ui.fyneApp.Preferences().Bool(settingNotificationsEnabled)
	notificationsEnabledCheck.OnChanged = func(b bool) { submit.Enable() }
	form.AppendItem(&widget.FormItem{
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
		cg.OnChanged = func(s []string) {
			submit.Enable()
		}
		form.AppendItem(widget.NewFormItem(c.String(), cg))
		groups = append(groups, cg)
	}
	content.Add(form)
	content.Add(container.NewPadded())

	submit.OnTapped = func() {
		w.ui.fyneApp.Preferences().SetBool(settingNotificationsEnabled, notificationsEnabledCheck.Checked)
		enabled := make([]string, 0)
		for _, cg := range groups {
			enabled = slices.Concat(enabled, cg.Selected)
		}
		w.ui.fyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, enabled)
		submit.Disable()
	}
	cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		w.window.Hide()
	})
	content.Add(container.NewHBox(layout.NewSpacer(), cancel, submit))

	return makePage("Notification settings", content)
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
