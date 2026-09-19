package xwidget

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestSnackbar_LifecycleAndTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := test.NewTempApp(t)

		window := app.NewWindow("Test Window")
		window.Resize(fyne.NewSize(400, 300))

		sb := NewSnackbar(window.Canvas())
		sb.Start()
		defer sb.Stop()

		// 1. Initial state check
		if sb.popup.Visible() {
			t.Fatal("expected snackbar popup to be hidden initially")
		}

		// 2. Queue a message and wait for it to process
		sb.Display("Hello, World!")
		synctest.Wait() // Wait for goroutines to process queue & call fyne.Do

		if !sb.popup.Visible() {
			t.Fatal("expected snackbar popup to be visible after Display")
		}

		// 3. Advance virtual time to trigger default timeout auto-dismiss
		time.Sleep(snackbarTimeoutDefault + 10*time.Millisecond)
		synctest.Wait()

		if sb.popup.Visible() {
			t.Fatal("expected snackbar popup to auto-hide after default timeout")
		}
	})
}

func TestSnackbar_CustomTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := test.NewTempApp(t)

		window := app.NewWindow("Test Window")
		window.Resize(fyne.NewSize(400, 300))

		sb := NewSnackbar(window.Canvas())
		sb.Start()
		defer sb.Stop()

		customTimeout := 500 * time.Millisecond
		sb.DisplayWithTimeout("Custom Timeout Message", customTimeout)
		synctest.Wait()

		if !sb.popup.Visible() {
			t.Fatal("expected snackbar popup to be visible")
		}

		// Advance time, but stay short of timeout
		time.Sleep(300 * time.Millisecond)
		synctest.Wait()
		if !sb.popup.Visible() {
			t.Fatal("expected snackbar popup to still be visible before timeout")
		}

		// Cross the custom timeout boundary
		time.Sleep(250 * time.Millisecond)
		synctest.Wait()

		if sb.popup.Visible() {
			t.Fatal("expected snackbar popup to hide after custom timeout")
		}
	})
}

func TestSnackbar_ManualDismissByTap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := test.NewTempApp(t)

		window := app.NewWindow("Test Window")
		window.Resize(fyne.NewSize(400, 300))

		sb := NewSnackbar(window.Canvas())
		sb.Start()
		defer sb.Stop()

		sb.Display("Tap me to dismiss")
		synctest.Wait()

		if !sb.popup.Visible() {
			t.Fatal("expected snackbar popup to be visible")
		}

		// Simulate user tapping the popup overlay
		sb.popup.Tapped(&fyne.PointEvent{})
		synctest.Wait()

		if sb.popup.Visible() {
			t.Fatal("expected snackbar popup to hide immediately after tap")
		}
	})
}

func TestSnackbar_SequentialQueueing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := test.NewTempApp(t)

		window := app.NewWindow("Test Window")
		window.Resize(fyne.NewSize(400, 300))

		sb := NewSnackbar(window.Canvas())
		sb.Start()
		defer sb.Stop()

		// Enqueue two messages
		sb.DisplayWithTimeout("Message 1", 200*time.Millisecond)
		sb.DisplayWithTimeout("Message 2", 200*time.Millisecond)

		synctest.Wait()
		if !sb.popup.Visible() {
			t.Fatal("expected first message to be shown")
		}

		// Wait for message 1 to time out
		time.Sleep(210 * time.Millisecond)
		synctest.Wait()

		// Message 2 should immediately take over and keep popup visible
		if !sb.popup.Visible() {
			t.Fatal("expected snackbar to show second message after first times out")
		}

		// Wait for message 2 to time out
		time.Sleep(210 * time.Millisecond)
		synctest.Wait()

		if sb.popup.Visible() {
			t.Fatal("expected snackbar to hide after second message completes")
		}
	})
}

func TestSnackbar_QueueingBeforeStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := test.NewTempApp(t)

		window := app.NewWindow("Test Window")

		sb := NewSnackbar(window.Canvas())

		// Messages queued while stopped should sit in the channel queue
		sb.DisplayWithTimeout("Queued Early", 100*time.Millisecond)
		synctest.Wait()

		if sb.popup.Visible() {
			t.Fatal("expected snackbar to remain hidden before Start() is called")
		}

		// Starting the snackbar should pick up queued messages
		sb.Start()
		defer sb.Stop()

		synctest.Wait()
		if !sb.popup.Visible() {
			t.Fatal("expected snackbar to display queued message after Start()")
		}

		time.Sleep(110 * time.Millisecond)
		synctest.Wait()

		if sb.popup.Visible() {
			t.Fatal("expected snackbar to hide after processing pre-queued item")
		}
	})
}

func TestSnackbar_StopAndRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := test.NewTempApp(t)

		window := app.NewWindow("Test Window")

		sb := NewSnackbar(window.Canvas())
		sb.Start()

		sb.DisplayWithTimeout("Going to stop", 500*time.Millisecond)
		synctest.Wait()

		if !sb.popup.Visible() {
			t.Fatal("expected snackbar to show active message")
		}

		// Stop aborts current context
		sb.Stop()
		synctest.Wait()

		// Ensure state resets properly
		if sb.isRunning.Load() {
			t.Fatal("expected isRunning to be false after Stop()")
		}

		// Restart and show new message
		sb.Start()
		defer sb.Stop()

		sb.DisplayWithTimeout("Restarted Message", 200*time.Millisecond)
		synctest.Wait()

		if !sb.popup.Visible() {
			t.Fatal("expected snackbar to function normally after restart")
		}
	})
}

func TestSnackbar_MessageContent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := test.NewTempApp(t)

		window := app.NewWindow("Test Window")
		window.Resize(fyne.NewSize(400, 300))

		sb := NewSnackbar(window.Canvas())
		sb.Start()
		defer sb.Stop()

		sb.Display("First message")
		synctest.Wait()

		if got, want := sb.text.String(), "First message"; got != want {
			t.Fatalf("expected displayed text %q, got %q", want, got)
		}

		time.Sleep(snackbarTimeoutDefault + 10*time.Millisecond)
		synctest.Wait()

		sb.Display("Second message")
		synctest.Wait()

		if got, want := sb.text.String(), "Second message"; got != want {
			t.Fatalf("expected displayed text %q, got %q", want, got)
		}
	})
}

func TestSnackbar_ConcurrentDisplay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := test.NewTempApp(t)

		window := app.NewWindow("Test Window")
		window.Resize(fyne.NewSize(400, 300))

		sb := NewSnackbar(window.Canvas())
		sb.Start()
		defer sb.Stop()

		const n = 10
		const timeout = 50 * time.Millisecond

		var wg sync.WaitGroup
		wg.Add(n)
		for i := range n {
			go func(i int) {
				defer wg.Done()
				sb.DisplayWithTimeout(fmt.Sprintf("message-%d", i), timeout)
			}(i)
		}
		wg.Wait()
		synctest.Wait()

		// Sample more often than the per-message timeout so no message's
		// display window can be skipped over, then keep sampling past the
		// point where all messages must have been processed.
		seen := make(map[string]bool)
		step := timeout / 3
		deadline := time.Duration(n)*timeout + timeout
		for elapsed := time.Duration(0); elapsed < deadline; elapsed += step {
			synctest.Wait()
			if sb.popup.Visible() {
				seen[sb.text.String()] = true
			}
			time.Sleep(step)
		}
		synctest.Wait()

		if sb.popup.Visible() {
			t.Fatal("expected popup hidden after processing all concurrently queued messages")
		}
		if len(seen) != n {
			t.Fatalf("expected %d distinct messages to be displayed, got %d: %v", n, len(seen), seen)
		}
	})
}

func TestSnackbar_TextWrappingCalculation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := test.NewTempApp(t)

		window := app.NewWindow("Test Window")
		window.Resize(fyne.NewSize(200, 300)) // Narrow canvas to force text wrapping

		sb := NewSnackbar(window.Canvas())
		sb.Start()
		defer sb.Stop()

		// Short text shouldn't trigger word wrap
		sb.DisplayWithTimeout("Hi", 100*time.Millisecond)
		synctest.Wait()
		if sb.text.Wrapping != fyne.TextWrapOff {
			t.Errorf("expected TextWrapOff for short text, got %v", sb.text.Wrapping)
		}

		time.Sleep(110 * time.Millisecond)
		synctest.Wait()

		// Very long text should trigger word wrap
		longText := "This is an extremely long message that will exceed the width of the canvas and force word wrapping"
		sb.DisplayWithTimeout(longText, 100*time.Millisecond)
		synctest.Wait()

		if sb.text.Wrapping != fyne.TextWrapWord {
			t.Errorf("expected TextWrapWord for long text, got %v", sb.text.Wrapping)
		}
	})
}
