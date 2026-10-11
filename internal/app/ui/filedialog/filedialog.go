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
	Go(job func(ctx context.Context)) bool
	IsDeveloperMode() bool
	IsMobile() bool
	MainWindow() fyne.Window
}

type ShowFileSaveWindowParams struct {
	CompletionText  string
	Extensions      []string
	Filename        string
	DisplaySnackbar func(string)
	Title           string
	WindowID        string // optional
	WriteFunc       func(ctx context.Context, w io.Writer) error
	Window          fyne.Window // required when running on mobile; used as the native picker's parent
}

func ShowSave(u baseUI, arg ShowFileSaveWindowParams) {
	if u.IsMobile() {
		d := createFileSaveDialog(u, arg, arg.Window)
		d.Show()
		return
	}

	w, _ := u.GetOrCreateWindow(arg.WindowID, arg.Title)
	d := createFileSaveDialog(u, arg, w)
	showDialogInWindow(u, w, d)
}

type ShowFileOpenWindowParams struct {
	CompletionText  string
	Extensions      []string
	DisplaySnackbar func(string)
	Title           string
	WindowID        string // optional
	ReadFunc        func(ctx context.Context, r io.Reader) error
	Window          fyne.Window // required when running on mobile; used as the native picker's parent
}

func ShowOpen(u baseUI, arg ShowFileOpenWindowParams) {
	if u.IsMobile() {
		d := createFileOpenDialog(u, arg, arg.Window)
		d.Show()
		return
	}

	w, _ := u.GetOrCreateWindow(arg.WindowID, arg.Title)
	d := createFileOpenDialog(u, arg, w)
	showDialogInWindow(u, w, d)
}

func showDialogInWindow(u baseUI, w fyne.Window, d *dialog.FileDialog) {
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
		handleSaveResult(u, arg, writer, err)
	}, w)
	if arg.Filename != "" {
		d.SetFileName(arg.Filename)
	}
	if len(arg.Extensions) > 0 {
		d.SetFilter(storage.NewExtensionFileFilter(arg.Extensions))
	}
	return d
}

func handleSaveResult(u baseUI, arg ShowFileSaveWindowParams, writer fyne.URIWriteCloser, err error) {
	if err != nil {
		arg.DisplaySnackbar("Error: " + u.ErrorDisplay(err))
		return
	}
	if writer == nil {
		return
	}
	ok := u.Go(func(ctx context.Context) {
		defer writer.Close()
		// Exports always finish, so no partial or empty file is left behind.
		err := arg.WriteFunc(context.WithoutCancel(ctx), writer)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			slog.Error(arg.Title, "error", err)
			fyne.Do(func() {
				arg.DisplaySnackbar("Error: " + u.ErrorDisplay(err))
			})
			return
		}
		fyne.Do(func() { arg.DisplaySnackbar(arg.CompletionText) })
	})
	if !ok {
		writer.Close()
	}
}

func createFileOpenDialog(u baseUI, arg ShowFileOpenWindowParams, w fyne.Window) *dialog.FileDialog {
	d := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		handleOpenResult(u, arg, reader, err)
	}, w)
	if len(arg.Extensions) > 0 {
		d.SetFilter(storage.NewExtensionFileFilter(arg.Extensions))
	}
	return d
}

func handleOpenResult(u baseUI, arg ShowFileOpenWindowParams, reader fyne.URIReadCloser, err error) {
	if err != nil {
		arg.DisplaySnackbar("Error: " + u.ErrorDisplay(err))
		return
	}
	if reader == nil {
		return
	}
	ok := u.Go(func(ctx context.Context) {
		defer reader.Close()
		err := arg.ReadFunc(ctx, reader)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			slog.Error(arg.Title, "error", err)
			fyne.Do(func() {
				arg.DisplaySnackbar("Error: " + u.ErrorDisplay(err))
			})
			return
		}
		fyne.Do(func() { arg.DisplaySnackbar(arg.CompletionText) })
	})
	if !ok {
		reader.Close()
	}
}
