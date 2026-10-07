package screens

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui/filedialog"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// runAsync runs f in a new goroutine.
//
// Tests replace it to run f synchronously.
var runAsync = func(f func()) { go f() }

// clearSelectsSilent clears the selection of filter chips without calling their OnChanged.
func clearSelectsSilent(chips ...*kxwidget.FilterChipSelect) {
	for _, c := range chips {
		c.Selected = ""
		c.Refresh()
	}
}

// latestRun identifies the newest of overlapping async runs,
// so results from older runs can be discarded.
type latestRun struct{ n atomic.Int64 }

// start begins a new run and returns a function reporting whether it is still the newest.
func (l *latestRun) start() func() bool {
	n := l.n.Add(1)
	return func() bool { return l.n.Load() == n }
}

// showWhenLoaded hides content behind a delayed spinner until load returns
// and returns the object to show in its place.
func showWhenLoaded(content fyne.CanvasObject, load func()) fyne.CanvasObject {
	content.Hide()
	spinner := kxwidget.NewSpinner()
	spinner.Hide()
	// needs Refresh after Show; Fyne won't repaint never-visible objects
	body := container.NewStack(
		content,
		container.NewCenter(container.NewStack(
			xwidget.NewSpacer(fyne.NewSquareSize(2*theme.IconInlineSize())),
			spinner,
		)),
	)
	var loaded bool
	// delay avoids the spinner flickering when loading is fast
	time.AfterFunc(100*time.Millisecond, func() {
		fyne.Do(func() {
			if loaded {
				return
			}
			spinner.Show()
			spinner.Start()
			body.Refresh()
		})
	})
	runAsync(func() {
		load()
		fyne.Do(func() {
			loaded = true
			spinner.Stop()
			spinner.Hide()
			content.Show()
			body.Refresh()
		})
	})
	return body
}

// copyRowsToClipboard copies rows from a data table to clipboard.
//
// The function can be called in the main thread.
func copyRowsToClipboard[T any](u baseUI, topic string, rows []T, transform func([]T) (string, error)) {
	rows2 := slices.Clone(rows)
	go func() {
		s, err := transform(rows2)
		if err != nil {
			slog.Error("Failed to copy to clipboard", "topic", topic, "error", err)
			u.DisplaySnackbar("ERROR: Failed to copy " + topic + " to clipboard")
			return
		}
		fyne.DoAndWait(func() {
			fyne.CurrentApp().Clipboard().SetContent(s)
		})
		u.DisplaySnackbar("Copied " + topic + " to clipboard")
	}()
}

// exportRowsAsCSV exports rows from a datatable to a CSV file.
//
// The function can be called in the main thread.
func exportRowsAsCSV[T any](u baseUI, topic string, filename string, rows []T, writeRows func(io.Writer, []T) error) {
	w := u.MainWindow()
	rows2 := slices.Clone(rows)
	filedialog.ShowSave(u, filedialog.ShowFileSaveWindowParams{
		CompletionText:  "Exported " + topic + " to CSV",
		Extensions:      []string{".csv"},
		Filename:        filename,
		DisplaySnackbar: u.DisplaySnackbar,
		Title:           "Export " + topic + " as CSV",
		WindowID:        fmt.Sprintf("export-%s-%s", topic, filename),
		WriteFunc: func(_ context.Context, w io.Writer) error {
			return writeRows(w, rows2)
		},
		Window: w,
	})
}

// showHelpPopUp shows a popUp with text as content
// and it's position aligned to widget obj.
func showHelpPopUp(text string, isMobile bool, obj fyne.CanvasObject) {
	var pu *widget.PopUp
	closePopUp := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		pu.Hide()
	})
	title := widget.NewLabel("Help")
	title.TextStyle.Bold = true
	body := widget.NewLabel(text)
	body.Wrapping = fyne.TextWrapWord

	p := theme.Padding()
	canvas := fyne.CurrentApp().Driver().CanvasForObject(obj)
	var spacerSize fyne.Size
	if isMobile {
		_, s := canvas.InteractiveArea()
		spacerSize = fyne.NewSize(s.Width-2*p, s.Height/2)
	} else {
		spacerSize = fyne.NewSize(300, 400)
	}
	spacer := xwidget.NewSpacer(spacerSize)
	c := container.NewStack(spacer, container.NewBorder(
		container.NewHBox(title, layout.NewSpacer(), closePopUp),
		nil,
		nil,
		nil,
		container.NewVScroll(container.NewPadded(body)),
	))
	pu = widget.NewPopUp(c, canvas)

	if isMobile {
		pos, s := canvas.InteractiveArea()
		x := pos.X
		y := pos.Y + s.Height/2
		pu.ShowAtPosition(fyne.NewPos(x, y))
	} else {
		x := obj.MinSize().Width - pu.MinSize().Width
		y := obj.MinSize().Height - pu.MinSize().Height + 2*p
		pu.ShowAtRelativePosition(fyne.NewPos(x, y), obj)
	}
}
