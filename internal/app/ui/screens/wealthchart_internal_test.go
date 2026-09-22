package screens

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWealthPalette(t *testing.T) {
	assert.GreaterOrEqual(t, len(wealthPalette), 12)
	seen := make(map[color.Color]bool)
	for _, c := range wealthPalette {
		assert.False(t, seen[c], "duplicate color in wealthPalette: %v", c)
		seen[c] = true
	}
}

func TestWealthArcColor(t *testing.T) {
	for i, c := range wealthPalette {
		assert.Equal(t, c, wealthArcColor(i))
	}
	// wraps around once the index exceeds the palette length
	assert.Equal(t, wealthPalette[0], wealthArcColor(len(wealthPalette)))
}

func TestWealthArcStyle(t *testing.T) {
	for i := range wealthPalette {
		style := wealthArcStyle(namedValue{}, i)
		assert.Equal(t, wealthArcColor(i), style.Fill.Color)
		assert.Equal(t, float32(1), style.Fill.Opacity)
	}
}

func TestWealthAxisValueFormatter(t *testing.T) {
	cases := []struct {
		value    float64
		expected string
	}{
		{1, "1.0"},
		{12.345, "12.3"},
		{0, "0.0"},
	}
	for _, tt := range cases {
		assert.Equal(t, tt.expected, wealthAxisValueFormatter(tt.value))
	}
}

func TestNewLegendEntry(t *testing.T) {
	test.NewTempApp(t)
	w := newLegendEntry("Assets", wealthBlueColor)
	r := w.CreateRenderer()
	require.NotNil(t, r)
}

func TestSeriesLegend_SetEntries(t *testing.T) {
	test.NewTempApp(t)
	w := newSeriesLegend()
	assert.Len(t, w.container.Objects, 0)

	entries := []*legendEntry{
		newLegendEntry("Assets", wealthBlueColor),
		newLegendEntry("Wallet", wealthOrangeColor),
	}
	w.SetEntries(entries...)

	assert.Len(t, w.container.Objects, 2)
}

func TestChartCard_NilLegend(t *testing.T) {
	test.NewTempApp(t)
	title := newChartTitleLabel()
	chart := canvas.NewRectangle(color.Black)
	w := newChartCard(title, nil, chart)

	r := w.CreateRenderer()

	require.NotNil(t, r)
}
