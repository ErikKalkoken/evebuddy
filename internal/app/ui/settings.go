package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/dustin/go-humanize"
)

// Setting keys
const (
	settingLastCharacterID              = "settings-lastCharacterID"
	settingMaxMails                     = "settings-maxMails"
	settingMaxMailsDefault              = 1_000
	settingMaxWalletTransactions        = "settings-maxWalletTransactions"
	settingMaxWalletTransactionsDefault = 10_000
	settingTheme                        = "settings-theme"
	settingThemeDefault                 = themeAuto
)

// Themes
const (
	themeAuto  = "Auto"
	themeDark  = "Dark"
	themeLight = "Light"
)

func (u *ui) showSettingsDialog() {
	d, err := makeSettingsDialog(u)
	if err != nil {
		t := "Failed to open the settings dialog"
		slog.Error(t, "err", err)
		u.showErrorDialog(t, err)
	} else {
		d.Show()
	}
}

func makeSettingsDialog(u *ui) (*dialog.CustomDialog, error) {
	maxMails := u.fyneApp.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault)
	maxMailsEntry := widget.NewEntry()
	maxMailsEntry.SetText(strconv.Itoa(maxMails))
	maxMailsEntry.Validator = newPositiveNumberValidator()

	maxTransactions := u.fyneApp.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
	maxTransactionsEntry := widget.NewEntry()
	maxTransactionsEntry.SetText(strconv.Itoa(maxTransactions))
	maxTransactionsEntry.Validator = newPositiveNumberValidator()

	clearBtn := widget.NewButton("Clear NOW", func() {
		d := dialog.NewConfirm(
			"Clear image cache",
			"Are you sure you want to clear the image cache?",
			func(confirmed bool) {
				if !confirmed {
					return
				}
				count, err := u.EveImageService.ClearCache()
				if err != nil {
					slog.Error(err.Error())
					u.showErrorDialog("Failed to clear image cache", err)
				} else {
					slog.Info("Cleared images cache", "count", count)
				}
			},
			u.window,
		)
		d.Show()
	})

	themeRadio := widget.NewRadioGroup(
		[]string{themeAuto, themeDark, themeLight}, func(s string) {},
	)
	name := u.fyneApp.Preferences().StringWithFallback(settingTheme, settingThemeDefault)
	themeRadio.SetSelected(name)

	var cacheSize string
	s, err := u.EveImageService.Size()
	if err != nil {
		cacheSize = "?"
	} else {
		cacheSize = humanize.Bytes(uint64(s))
	}
	cacheHintText := fmt.Sprintf("Clear the local image cache (%s)", cacheSize)
	var d *dialog.CustomDialog
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
			{
				Text:     "Theme",
				Widget:   themeRadio,
				HintText: "Chose the preferred theme",
			}, {
				Text:     "Image cache",
				Widget:   clearBtn,
				HintText: cacheHintText,
			},
		},
		OnSubmit: func() {
			maxMails, err := strconv.Atoi(maxMailsEntry.Text)
			if err != nil {
				return
			}
			u.fyneApp.Preferences().SetInt(settingMaxMails, maxMails)
			u.themeSet(themeRadio.Selected)
			u.fyneApp.Preferences().SetString(settingTheme, themeRadio.Selected)
			d.Hide()
		},
		OnCancel: func() {
			d.Hide()
		},
	}
	d = dialog.NewCustomWithoutButtons("Settings", form, u.window)
	return d, nil
}

// // settingsArea is the UI area for settings.
// type settingsArea struct {
// 	content *fyne.Container
// 	ui      *ui
// }

// func (u *ui) newSettingsArea() *settingsArea {
// 	content := container.NewVBox()
// 	m := &settingsArea{
// 		ui:      u,
// 		content: content,
// 	}
// 	return m
// }

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
