// Package filedialog shows file dialogs, in a window on desktop and natively on mobile.
package filedialog

import (
	"context"
	"io"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

type baseUI interface {
	ErrorDisplay(err error) string
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	IsDeveloperMode() bool
	IsMobile() bool
	MainWindow() fyne.Window
}

type ShowFileSaveWindowParams struct {
	CompletionText string
	Extensions     []string
	Filename       string
	ShowSnackbar   func(string)
	Title          string
	WindowID       string // optional
	WriteFunc      func(ctx context.Context, w io.Writer) error
	Window         fyne.Window // required when running on mobile; used as the native picker's parent
}

func ShowSave(u baseUI, arg ShowFileSaveWindowParams) {
	if u.IsMobile() {
		d := createFileSaveDialog(u, arg, arg.Window)
		d.Show()
		return
	}

	w, _ := u.GetOrCreateWindow(arg.WindowID, arg.Title)

	d := createFileSaveDialog(u, arg, w)

	_, s := u.MainWindow().Canvas().InteractiveArea()
	winSize := fyne.NewSize(s.Width*0.8, s.Height*0.8)
	d.SetOnClosed(func() {
		w.Close()
	})
	w.Resize(winSize)
	d.Show()
	d.Resize(winSize)
	w.SetFixedSize(true)
	w.Show()
}

func createFileSaveDialog(u baseUI, arg ShowFileSaveWindowParams, w fyne.Window) *dialog.FileDialog {
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			arg.ShowSnackbar("Error: " + u.ErrorDisplay(err))
			return
		}
		if writer == nil {
			return
		}
		go func() {
			defer writer.Close()
			err := arg.WriteFunc(context.Background(), writer)
			if err != nil {
				slog.Error(arg.Title, "error", err)
				fyne.Do(func() {
					arg.ShowSnackbar("Error: " + u.ErrorDisplay(err))
				})
				return
			}
			fyne.Do(func() { arg.ShowSnackbar(arg.CompletionText) })
		}()
	}, w)
	d.SetFileName(arg.Filename)
	if len(arg.Extensions) > 0 {
		d.SetFilter(storage.NewExtensionFileFilter(arg.Extensions))
	}
	return d
}
