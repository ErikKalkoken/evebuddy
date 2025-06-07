package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/widget"
)

func TestFilterChip_EnabledOff(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChip("Test", nil)
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()

	test.AssertImageMatches(t, "filterchip/enabled_off.png", w.Canvas().Capture())
}

func TestFilterChip_DisabledOff(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChip("Test", nil)
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()
	chip.Disable()

	test.AssertImageMatches(t, "filterchip/disabled_off.png", w.Canvas().Capture())
}

func TestFilterChip_EnabledOn(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChip("Test", nil)
	chip.On = true
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()

	test.AssertImageMatches(t, "filterchip/enabled_on.png", w.Canvas().Capture())
}

func TestFilterChip_DisabledOn(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	chip := widget.NewFilterChip("Test", nil)
	chip.On = true
	chip.Disable()
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()

	test.AssertImageMatches(t, "filterchip/disabled_on.png", w.Canvas().Capture())
}

func TestFilterChip_CanSwitchOnWhenEnabled(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	var tapped bool
	chip := widget.NewFilterChip("Test", func(on bool) {
		tapped = true
	})
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()
	w.Resize(fyne.NewSize(150, 50))

	test.Tap(chip)
	assert.True(t, tapped)
	test.AssertImageMatches(t, "filterchip/tapped_enabled_on.png", w.Canvas().Capture())
}

func TestFilterChip_CanSwitchOffWhenEnabled(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	var tapped bool
	chip := widget.NewFilterChip("Test", func(on bool) {
		tapped = true
	})
	chip.On = true
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()
	w.Resize(fyne.NewSize(150, 50))

	test.Tap(chip)
	assert.True(t, tapped)
	test.AssertImageMatches(t, "filterchip/tapped_enabled_off.png", w.Canvas().Capture())
}

func TestFilterChip_CanNotSwitchWhenDisabledOff(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	var tapped bool
	chip := widget.NewFilterChip("Test", func(on bool) {
		tapped = true
	})
	chip.Disable()
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()

	test.Tap(chip)
	assert.False(t, tapped)
	test.AssertImageMatches(t, "filterchip/disabled_off.png", w.Canvas().Capture())
}

func TestFilterChip_CanNotSwitchWhenDisabledOn(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	var tapped bool
	chip := widget.NewFilterChip("Test", func(on bool) {
		tapped = true
	})
	chip.On = true
	chip.Disable()
	w := test.NewWindow(container.NewCenter(chip))
	defer w.Close()

	test.Tap(chip)
	assert.False(t, tapped)
	test.AssertImageMatches(t, "filterchip/disabled_on.png", w.Canvas().Capture())
}
