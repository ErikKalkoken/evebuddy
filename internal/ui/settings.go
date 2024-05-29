package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
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
	maxMails, ok, err := u.service.DictionaryInt(model.SettingMaxMails)
	if err != nil {
		return nil, err
	}
	if !ok {
		maxMails = model.SettingMaxMailsDefault
	}
	maxMailsEntry := widget.NewEntry()
	maxMailsEntry.SetText(strconv.Itoa(int(maxMails)))
	maxMailsEntry.Validator = newPositiveNumberValidator()

	clearBtn := widget.NewButton("Clear NOW", func() {
		d := dialog.NewConfirm(
			"Clear image cache",
			"Are you sure you want to clear the image cache?",
			func(confirmed bool) {
				if !confirmed {
					return
				}
				count, err := u.imageManager.Clear()
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
		[]string{model.ThemeAuto, model.ThemeDark, model.ThemeLight}, func(s string) {},
	)
	name, ok, err := u.service.DictionaryString(model.SettingTheme)
	if err == nil && ok {
		themeRadio.SetSelected(name)
	}

	var cacheSize string
	s, err := u.imageManager.Size()
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
				HintText: "Maximum number of mails downloaded from the server",
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
			u.service.DictionarySetInt(model.SettingMaxMails, maxMails)
			u.setTheme(themeRadio.Selected)
			u.service.DictionarySetString(model.SettingTheme, themeRadio.Selected)
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

func newPositiveNumberValidator() fyne.StringValidator {
	myErr := errors.New("must be positive number")
	return func(text string) error {
		val, err := strconv.Atoi(text)
		if err != nil {
			return myErr
		}
		if val <= 0 {
			return myErr
		}
		return nil
	}
}
