package widget

import (
	"context"
	"log/slog"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/syncqueue"
)

// progressModalTask represents a task that is executed by a [ProgressModal].
type progressModalTask struct {
	message string
	action  func()
}

// A ProgressModal shows a modal dialog while an action is running
// and disappear once the action is completed.
//
// ProgressModal are designed to be created once for each window and then re-used. The can be used concurrently.
//
// When a ProgressModal receives several tasks at the same time, it will queue them and execute them one after the other.
type ProgressModal struct {
	button    *widget.Button
	hideC     chan struct{}
	dialog    dialog.Dialog
	isRunning atomic.Bool
	label     *widget.Label
	pb        *widget.ProgressBarInfinite
	q         *syncqueue.SyncQueue[progressModalTask]
	stopC     chan struct{}
}

// NewProgressModal returns a new progress modal. Call Start() to activate it.
func NewProgressModal(win fyne.Window) *ProgressModal {
	m := &ProgressModal{
		hideC: make(chan struct{}),
		label: widget.NewLabel(""),
		pb:    widget.NewProgressBarInfinite(),
		q:     syncqueue.New[progressModalTask](),
		stopC: make(chan struct{}),
	}
	m.button = widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
		m.hideC <- struct{}{}
	})
	m.dialog = dialog.NewCustomWithoutButtons(
		"Please wait",
		container.NewVBox(m.label, container.NewBorder(nil, nil, nil, m.button, m.pb)),
		win,
	)
	return m
}

// Execute executes a task while the progress modal is shown.
// Each task is running in it's own Goroutine.
func (m *ProgressModal) Execute(message string, action func()) {
	m.q.Put(progressModalTask{message: message, action: action})
}

// Start starts the task execution and should be called after the Fyne app is started.
func (m *ProgressModal) Start() {
	isRunning := !m.isRunning.CompareAndSwap(false, true)
	if isRunning {
		slog.Warn("ProgressModal has already been started")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-m.stopC
		cancel()
	}()
	go func() {
	L:
		for {
			task, err := m.q.Get(ctx)
			if err != nil {
				break
			}
			fyne.DoAndWait(func() {
				m.pb.Start()
				m.label.SetText(task.message)
				m.dialog.Show()
			})
			defer fyne.Do(func() {
				m.dialog.Hide()
			})
			done := make(chan struct{})
			go func() {
				task.action()
				done <- struct{}{}
			}()
			select {
			case <-m.stopC:
				cancel()
				break L
			case <-m.hideC:
			case <-done:
			}
			fyne.Do(func() {
				m.dialog.Hide()
			})
		}
		m.isRunning.Store(false)
		slog.Debug("ProgressModal stopped")
	}()
	slog.Debug("ProgressModal started")
}

// Stop stops a running [ProgressModal] and allows the gc to clean up it's resources.
func (m *ProgressModal) Stop() {
	if !m.isRunning.Load() {
		return
	}
	m.stopC <- struct{}{}
}

// IsRunning reports whether a [ProgressModal] has been started.
func (m *ProgressModal) IsRunning() bool {
	return m.isRunning.Load()
}
