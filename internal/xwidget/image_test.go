package xwidget_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNewImageFromResource(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	img := xwidget.NewImageFromResource(theme.HomeIcon(), fyne.NewSize(50, 60))

	assert.Equal(t, canvas.ImageFillContain, img.FillMode)
	assert.Equal(t, fyne.NewSize(50, 60), img.MinSize())
	assert.Equal(t, theme.HomeIcon(), img.Resource)
}

func TestLoadResourceAsyncWithCache_CacheHit(t *testing.T) {
	test.NewTempApp(t)

	cached := theme.HomeIcon()
	var got fyne.Resource

	xwidget.LoadResourceAsyncWithCache(
		theme.CancelIcon(),
		func() (fyne.Resource, bool) { return cached, true },
		func(r fyne.Resource) { got = r },
		func() (fyne.Resource, error) {
			t.Fatal("loader should not be called on cache hit")
			return nil, nil
		},
		func(fyne.Resource) {
			t.Fatal("setter should not be called on cache hit")
		},
	)

	assert.Equal(t, cached, got)
}

func TestLoadResourceAsyncWithCache_CacheMissLoadsAsync(t *testing.T) {
	test.NewTempApp(t)

	initial := theme.CancelIcon()
	loaded := theme.HomeIcon()

	var mu sync.Mutex
	var updates []fyne.Resource
	var setResource fyne.Resource
	done := make(chan struct{})

	xwidget.LoadResourceAsyncWithCache(
		initial,
		func() (fyne.Resource, bool) { return nil, false },
		func(r fyne.Resource) {
			mu.Lock()
			updates = append(updates, r)
			n := len(updates)
			mu.Unlock()
			if n == 2 {
				close(done)
			}
		},
		func() (fyne.Resource, error) { return loaded, nil },
		func(r fyne.Resource) { setResource = r },
	)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async load to complete")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []fyne.Resource{initial, loaded}, updates)
	assert.Equal(t, loaded, setResource)
}

func TestLoadResourceAsyncWithCache_LoaderErrorFallsBackToBrokenImage(t *testing.T) {
	test.NewTempApp(t)

	initial := theme.CancelIcon()
	done := make(chan struct{})
	var final fyne.Resource

	xwidget.LoadResourceAsyncWithCache(
		initial,
		func() (fyne.Resource, bool) { return nil, false },
		func(r fyne.Resource) {
			final = r
			if r != initial {
				close(done)
			}
		},
		func() (fyne.Resource, error) { return nil, errors.New("boom") },
		func(fyne.Resource) {},
	)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async load to complete")
	}

	assert.Equal(t, theme.BrokenImageIcon(), final)
}
