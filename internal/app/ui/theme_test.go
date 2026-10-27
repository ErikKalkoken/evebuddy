package ui_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestTheme(t *testing.T) {
	test.NewTempApp(t)

	t.Run("auto mode uses requested variant", func(t *testing.T) {
		// given
		th := ui.New(theme.DefaultTheme(), settings.Auto)
		// when
		dark := th.Color(ui.ColorNameInfo, theme.VariantDark)
		light := th.Color(ui.ColorNameInfo, theme.VariantLight)
		// then
		assert.NotEqual(t, dark, light)
	})
	t.Run("dark mode always uses dark variant", func(t *testing.T) {
		// given
		auto := ui.New(theme.DefaultTheme(), settings.Auto)
		th := ui.New(theme.DefaultTheme(), settings.Dark)
		// when
		got := th.Color(ui.ColorNameInfo, theme.VariantLight)
		// then
		xassert.Equal(t, auto.Color(ui.ColorNameInfo, theme.VariantDark), got)
	})
	t.Run("light mode always uses light variant", func(t *testing.T) {
		// given
		auto := ui.New(theme.DefaultTheme(), settings.Auto)
		th := ui.New(theme.DefaultTheme(), settings.Light)
		// when
		got := th.Color(ui.ColorNameInfo, theme.VariantDark)
		// then
		xassert.Equal(t, auto.Color(ui.ColorNameInfo, theme.VariantLight), got)
	})
	t.Run("fixed mode also applies to default colors", func(t *testing.T) {
		// given
		th := ui.New(theme.DefaultTheme(), settings.Dark)
		// when
		got := th.Color(theme.ColorNameBackground, theme.VariantLight)
		// then
		xassert.Equal(t, theme.DefaultTheme().Color(theme.ColorNameBackground, theme.VariantDark), got)
	})
	for _, c := range []fyne.ThemeColorName{
		ui.ColorNameInfo,
		ui.ColorNameCreative,
		ui.ColorNameSystem,
		ui.ColorNameAttention,
	} {
		for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
			t.Run(fmt.Sprintf("provides custom color %s for variant %d", c, v), func(t *testing.T) {
				// given
				th := ui.New(theme.DefaultTheme(), settings.Auto)
				// when/then
				assert.NotNil(t, th.Color(c, v))
			})
		}
	}
	t.Run("provides small text size", func(t *testing.T) {
		// given
		th := ui.New(theme.DefaultTheme(), settings.Auto)
		// when/then
		xassert.Equal(t, float32(12), th.Size(ui.SizeNameSmallText))
	})
	t.Run("uses default theme for other sizes, fonts and icons", func(t *testing.T) {
		// given
		def := theme.DefaultTheme()
		th := ui.New(def, settings.Auto)
		// when/then
		xassert.Equal(t, def.Size(theme.SizeNamePadding), th.Size(theme.SizeNamePadding))
		xassert.Equal(t, def.Font(fyne.TextStyle{Bold: true}), th.Font(fyne.TextStyle{Bold: true}))
		xassert.Equal(t, def.Icon(theme.IconNameHome), th.Icon(theme.IconNameHome))
	})
}
