package screens

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui/filedialog"
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
	rows2 := slices.Clone(rows)
	filedialog.ShowSave(u, filedialog.ShowFileSaveWindowParams{
		CompletionText: "Exported " + topic + " to CSV",
		Extensions:     []string{".csv"},
		Filename:       filename,
		ShowSnackbar:   u.ShowSnackbar,
		Title:          "Export " + topic + " as CSV",
		WindowID:       fmt.Sprintf("export-%s-%s", topic, filename),
		WriteFunc: func(_ context.Context, w io.Writer) error {
			return writeRows(w, rows2)
		},
		Window: w,
	})
}
