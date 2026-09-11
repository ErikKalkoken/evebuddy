package xwidget

import (
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
)

func TestProgressButton_InitialState(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	pb := NewProgressButton("Click", theme.HomeIcon(), nil)
	w := test.NewWindow(pb)
	defer w.Close()

	assert.False(t, pb.progress.Visible())
	assert.False(t, pb.button.locked)
}

func TestProgressButton_TapRunsActionAndRestoresState(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	started := make(chan struct{})
	proceed := make(chan struct{})
	var ran atomic.Bool

	pb := NewProgressButton("Click", theme.HomeIcon(), func() {
		ran.Store(true)
		close(started)
		<-proceed
	})
	w := test.NewWindow(pb)
	defer w.Close()

	test.Tap(pb.button)
	<-started

	assert.True(t, pb.button.locked)
	assert.Equal(t, "", pb.button.Text)
	assert.True(t, pb.progress.Visible())

	close(proceed)

	assert.Eventually(t, func() bool {
		return !pb.button.locked
	}, time.Second, 5*time.Millisecond)
	assert.True(t, ran.Load())
	assert.False(t, pb.progress.Visible())
	assert.Equal(t, "Click", pb.button.Text)
}

func TestProgressButton_TapIgnoredWhenNoAction(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	pb := NewProgressButton("Click", theme.HomeIcon(), nil)
	w := test.NewWindow(pb)
	defer w.Close()

	assert.NotPanics(t, func() { test.Tap(pb.button) })
	assert.False(t, pb.button.locked)
}

func TestProgressButton_SecondTapWhileRunningIsIgnored(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	started := make(chan struct{})
	proceed := make(chan struct{})
	var runCount atomic.Int32

	pb := NewProgressButton("Click", theme.HomeIcon(), func() {
		runCount.Add(1)
		close(started)
		<-proceed
	})
	w := test.NewWindow(pb)
	defer w.Close()

	test.Tap(pb.button)
	<-started
	test.Tap(pb.button) // no-op: button is locked while running

	close(proceed)
	assert.Eventually(t, func() bool { return !pb.button.locked }, time.Second, 5*time.Millisecond)
	assert.EqualValues(t, 1, runCount.Load())
}

func TestProgressButton_SetTextIconImportanceWhileIdle(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	pb := NewProgressButton("Click", theme.HomeIcon(), nil)
	w := test.NewWindow(pb)
	defer w.Close()

	pb.SetText("New")
	assert.Equal(t, "New", pb.button.Text)

	pb.SetIcon(theme.CancelIcon())
	assert.Equal(t, theme.CancelIcon(), pb.button.Icon)

	pb.SetImportance(widget.HighImportance)
	assert.Equal(t, widget.HighImportance, pb.button.Importance)
}

func TestProgressButton_SetTextIconWhileRunningIsDeferredUntilCompletion(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	started := make(chan struct{})
	proceed := make(chan struct{})

	pb := NewProgressButton("Click", theme.HomeIcon(), func() {
		close(started)
		<-proceed
	})
	w := test.NewWindow(pb)
	defer w.Close()

	test.Tap(pb.button)
	<-started

	pb.SetText("Later")
	pb.SetIcon(theme.CancelIcon())
	assert.NotEqual(t, "Later", pb.button.Text) // button is cleared while running
	assert.Equal(t, "Later", pb.label)          // pending value stored for restoration

	close(proceed)

	assert.Eventually(t, func() bool { return !pb.button.locked }, time.Second, 5*time.Millisecond)
	assert.Equal(t, "Later", pb.button.Text)
	assert.Equal(t, theme.CancelIcon(), pb.button.Icon)
}

func TestProgressButton_DisableWhileRunningAppliesAfterCompletion(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	started := make(chan struct{})
	proceed := make(chan struct{})

	pb := NewProgressButton("Click", theme.HomeIcon(), func() {
		close(started)
		<-proceed
	})
	w := test.NewWindow(pb)
	defer w.Close()

	test.Tap(pb.button)
	<-started

	pb.Disable()
	assert.True(t, pb.Disabled())
	assert.False(t, pb.button.Disabled()) // underlying button not disabled yet

	close(proceed)

	assert.Eventually(t, func() bool { return !pb.button.locked }, time.Second, 5*time.Millisecond)
	assert.True(t, pb.button.Disabled())
	assert.True(t, pb.Disabled())
}
