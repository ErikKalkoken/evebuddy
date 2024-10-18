package uninstall

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

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/appdirs"
)

var ErrCancel = errors.New("user aborted")

type UI struct {
	app    fyne.App
	ad     appdirs.AppDirs
	window fyne.Window
}

func NewUI(fyneApp fyne.App, ad appdirs.AppDirs) UI {
	w := fyneApp.NewWindow("Uninstall - EVE Buddy")
	x := UI{
		app:    fyneApp,
		ad:     ad,
		window: w,
	}
	return x
}

// RunApp runs the uninstall app
func (u *UI) ShowAndRun() {
	c := u.makePage()
	u.window.SetContent(c)
	u.window.Resize(fyne.Size{Width: 400, Height: 344})
	u.window.ShowAndRun()
}

func (u *UI) makePage() *fyne.Container {
	label := widget.NewLabel(fmt.Sprintf(
		"Are you sure you want to uninstall %s\n"+
			"and delete all user files?",
		u.app.Metadata().Name,
	))
	okBtn := widget.NewButtonWithIcon("Uninstall", theme.ConfirmIcon(), func() {
		title := widget.NewLabel("Removing user files...")
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
			if err := u.removeFolders(ctx, pb); err == ErrCancel {
				title.SetText("Uninstall aborted")
			} else if err != nil {
				title.SetText("Uninstall failed")
				errText.SetText(fmt.Sprintf("ERROR: %s", err))
			} else {
				title.SetText("Uninstall completed")
			}
			cancel()
			cancelBtn.Disable()
			closeBtn.Enable()
		}()
	})
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		u.closeWithDialog("Aborted")
	})
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
	d := dialog.NewInformation("Uninstall", message, u.window)
	d.SetOnClosed(u.window.Close)
	d.Show()
}

func (u *UI) removeFolders(ctx context.Context, pb *widget.ProgressBar) error {
	folders := []string{u.ad.Log, u.ad.Cache, u.ad.Data}
	for i, p := range folders {
		select {
		case <-ctx.Done():
			return ErrCancel
		default:
			if err := os.RemoveAll(p); err != nil {
				return err
			}
			pb.SetValue(float64(i+1) / float64(len(folders)))
			slog.Info("Deleted directory", "path", p)
		}
	}
	for _, k := range ui.SettingKeys() {
		u.app.Preferences().RemoveValue(k)
		slog.Info("Deleted setting", "key", k)
	}
	return nil
}
