// Package deleteapp contains the Fyne app for deleting the current users's data.
package deleteapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
)

var ErrCancel = errors.New("user aborted")

type UI struct {
	DataDir string

	app    fyne.App
	window fyne.Window
}

func NewUI(fyneApp fyne.App) UI {
	w := fyneApp.NewWindow("Delete User Data - EVE Buddy")
	x := UI{
		app:    fyneApp,
		window: w,
	}
	return x
}

// ShowAndRun runs the delete data app
func (u *UI) ShowAndRun() {
	c := u.makePage()
	u.window.SetContent(c)
	u.window.Resize(fyne.Size{Width: 400, Height: 200})
	u.window.ShowAndRun()
}

func (u *UI) makePage() *fyne.Container {
	okBtn := widget.NewButtonWithIcon("Delete", theme.ConfirmIcon(), func() {
		title := widget.NewLabel("Deleting user data...")
		pb := widget.NewProgressBar()
		errText := widget.NewLabel("")
		errText.Importance = widget.DangerImportance
		ctx, cancel := context.WithCancel(context.Background())
		cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
			cancel()
			u.closeWithDialog("Aborted")
		})
		closeBtn := widget.NewButtonWithIcon("Close", theme.ConfirmIcon(), func() {
			u.window.Close()
		})
		closeBtn.Disable()
		c := container.NewBorder(
			nil,
			container.NewHBox(cancelBtn, layout.NewSpacer(), closeBtn),
			nil,
			nil,
			container.NewVBox(title, pb, errText),
		)
		u.window.SetContent(c)
		go func() {
			RemoveSettings(u.app)
			if err := RemoveFolders(ctx, u.DataDir, func(p float64) {
				fyne.Do(func() {
					pb.SetValue(p)
				})
			}); err == ErrCancel {
				fyne.Do(func() {
					title.SetText("Data delete aborted")
				})
			} else if err != nil {
				fyne.Do(func() {
					title.SetText("Data delete failed")
					errText.SetText(fmt.Sprintf("ERROR: %s", err))
				})
			} else {
				fyne.Do(func() {
					title.SetText("Data delete completed")
				})
			}
			cancel()
			fyne.Do(func() {
				cancelBtn.Disable()
				closeBtn.Enable()
			})
		}()
	})
	okBtn.Importance = widget.DangerImportance
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		u.closeWithDialog("Aborted")
	})
	label := widget.NewLabel(fmt.Sprint(
		"Are you sure you want to delete\n" +
			"all data of the current user?",
	))
	c := container.NewBorder(
		nil,
		container.NewHBox(cancelBtn, layout.NewSpacer(), okBtn),
		nil,
		nil,
		container.NewCenter(label),
	)
	return c
}

func (u *UI) closeWithDialog(message string) {
	d := dialog.NewInformation("Delete User Data", message, u.window)
	d.SetOnClosed(u.window.Close)
	d.Show()
}

func RemoveFolders(ctx context.Context, dir string, update func(p float64)) error {
	folders := []string{dir}
	for i, p := range folders {
		select {
		case <-ctx.Done():
			return ErrCancel
		default:
			if err := os.RemoveAll(p); err != nil {
				return err
			}
			slog.Info("Deleted directory", "path", p)
			if update != nil {
				update(float64(i+1) / float64(len(folders)))
			}
		}
	}
	return nil
}

func RemoveSettings(app fyne.App) {
	keys := set.Of(settings.Keys()...)
	for k := range keys.All() {
		app.Preferences().RemoveValue(k)
	}
	slog.Info("Deleted setting keys", "count", keys.Size())
}
