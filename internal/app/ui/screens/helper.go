package screens

import (
	"io"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

// copyRowsToClipboard copies rows from a data table to clipboard.
//
// The function can be called in the main thread.
func copyRowsToClipboard[T any](u baseUI, topic string, rows []T, transform func([]T) (string, error)) {
	rows2 := slices.Clone(rows)
	go func() {
		s, err := transform(rows2)
		if err != nil {
			slog.Error("Failed to copy to clipboard", "topic", topic, "error", err)
			u.ShowSnackbar("ERROR: Failed to copy " + topic + " to clipboard")
			return
		}
		fyne.DoAndWait(func() {
			fyne.CurrentApp().Clipboard().SetContent(s)
		})
		u.ShowSnackbar("Copied " + topic + " to clipboard")
	}()
}

// exportRowsAsCSV exports rows from a datatable to a CSV file.
//
// The function can be called in the main thread.
func exportRowsAsCSV[T any](u baseUI, topic string, filename string, rows []T, writeRows func(io.Writer, []T) error) {
	w := u.MainWindow()
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil {
			return
		}
		showError := func(err error) {
			ui.ShowErrorAndLog("Failed to export "+topic, err, u.IsDeveloperMode(), w)
		}
		if err != nil {
			writer.Close()
			showError(err)
			return
		}
		rows2 := slices.Clone(rows)
		go func() {
			defer writer.Close()
			err := writeRows(writer, rows2)
			if err != nil {
				fyne.Do(func() {
					showError(err)
				})
				return
			}
			u.ShowSnackbar("Exported " + topic + " to CSV")
		}()
	}, w)

	d.SetFileName(filename)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
	d.SetTitleText("Export " + topic + " as CSV")
	d.Show()

	_, s := w.Canvas().InteractiveArea()
	winSize := fyne.NewSize(s.Width*0.8, s.Height*0.8)
	d.Resize(winSize)
}
