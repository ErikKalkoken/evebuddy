package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestName(t *testing.T) {
	t.Run("falls back to default name without metadata", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		// when/then
		xassert.Equal(t, "EVE Buddy", ui.Name())
	})
}

func TestWebsiteRootURL(t *testing.T) {
	t.Run("falls back to default URL without metadata", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		// when/then
		xassert.Equal(t, "https://github.com/ErikKalkoken/evebuddy", ui.WebsiteRootURL().String())
	})
}

func TestSkillDisplayName(t *testing.T) {
	xassert.Equal(t, "Gunnery III", ui.SkillDisplayName("Gunnery", 3))
}

func TestFormatISKAmount(t *testing.T) {
	cases := []struct {
		name string
		v    float64
		want string
	}{
		{"small amount", 500, "500.00 ISK"},
		{"largest amount without long form", 999, "999.00 ISK"},
		{"large amount with long form", 1234567.891, "1,234,567.89 ISK (1.23 M)"},
		{"negative amount with long form", -1234567.891, "-1,234,567.89 ISK (-1.23 M)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xassert.Equal(t, tc.want, ui.FormatISKAmount(tc.v))
		})
	}
}

func TestMakeEveEntityActionLabel(t *testing.T) {
	test.NewTempApp(t)

	t.Run("shows link that runs action for entity", func(t *testing.T) {
		// given
		var got []*app.EveEntity
		ee := &app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		x := ui.MakeEveEntityActionLabel(ee, func(o *app.EveEntity) {
			got = append(got, o)
		})
		link, ok := x.(*widget.Hyperlink)
		require.True(t, ok)
		// when
		test.Tap(link)
		// then
		xassert.Equal(t, "Bruce Wayne", link.Text)
		xassert.Equal(t, []*app.EveEntity{ee}, got)
	})
	t.Run("shows placeholder for nil", func(t *testing.T) {
		// when
		x := ui.MakeEveEntityActionLabel(nil, func(_ *app.EveEntity) {})
		// then
		label, ok := x.(*widget.Label)
		require.True(t, ok)
		xassert.Equal(t, "-", label.Text)
	})
	t.Run("shows link that runs action for character", func(t *testing.T) {
		// given
		var got []*app.EveEntity
		x := ui.MakeCharacterActionLabel(42, "Bruce Wayne", func(o *app.EveEntity) {
			got = append(got, o)
		})
		link, ok := x.(*widget.Hyperlink)
		require.True(t, ok)
		// when
		test.Tap(link)
		// then
		xassert.Equal(t, []*app.EveEntity{{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter}}, got)
	})
}

func TestMakeLocationLabel(t *testing.T) {
	test.NewTempApp(t)

	t.Run("shows link that shows location", func(t *testing.T) {
		// given
		var got []int64
		x := ui.MakeLocationLabel(&app.EveLocationShort{ID: 60003760, Name: optional.New("Jita IV - Moon 4")}, func(id int64) {
			got = append(got, id)
		})
		link, ok := x.(*widget.Hyperlink)
		require.True(t, ok)
		// when
		test.Tap(link)
		// then
		xassert.Equal(t, "Jita IV - Moon 4", link.Text)
		xassert.Equal(t, []int64{60003760}, got)
	})
	t.Run("shows placeholder for nil", func(t *testing.T) {
		// when
		x := ui.MakeLocationLabel(nil, func(_ int64) {})
		// then
		label, ok := x.(*widget.Label)
		require.True(t, ok)
		xassert.Equal(t, "?", label.Text)
	})
}

func TestMakeDetailWindow(t *testing.T) {
	test.NewTempApp(t)

	t.Run("panics without window", func(t *testing.T) {
		assert.Panics(t, func() {
			ui.MakeDetailWindow(ui.MakeDetailWindowParams{Title: "Title"})
		})
	})
	t.Run("loads image and runs image action when tapped", func(t *testing.T) {
		// given
		w := test.NewWindow(nil)
		defer w.Close()
		var tapped int
		ui.MakeDetailWindow(ui.MakeDetailWindowParams{
			Content:     widget.NewLabel("Content"),
			ImageAction: func() { tapped++ },
			ImageLoader: func(setter func(r fyne.Resource)) {
				setter(icons.Characterplaceholder64Jpeg)
			},
			Title:  "Title",
			Window: w,
		})
		var image *xwidget.TappableImage
		for _, o := range test.LaidOutObjects(w.Content()) {
			if x, ok := o.(*xwidget.TappableImage); ok {
				image = x
			}
		}
		require.NotNil(t, image)
		// when
		test.Tap(image)
		// then
		xassert.Equal(t, 1, tapped)
	})
}

// TestShakedown renders a window with all widgets of this package to detect panics.
func TestShakedown(t *testing.T) {
	a := test.NewTempApp(t)
	a.Settings().SetTheme(ui.New(a.Settings().Theme(), settings.Auto))
	w := test.NewWindow(nil)
	defer w.Close()
	eis := testutil.NewEveImageServiceStub()
	iv := &testdouble.InfoViewerFake{}

	entity := ui.NewEveEntityListItem(eis.EveEntityLogoAsync)
	entity.IsAvatar = true
	entity.Set(&app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
	total := ui.NewEveEntityListItem(eis.EveEntityLogoAsync)
	total.Set(&app.EveEntity{Name: "Total"})
	link := ui.NewInfoLink(iv)
	link.SetRace(&app.EveRace{ID: 1, Name: "Caldari"})
	attributes := ui.NewAttributeList(
		ui.AttributeItem{Label: "Section", IsHeading: true},
		ui.AttributeItem{Label: "Alpha", Value: "1", InfoAction: func() {}},
		ui.AttributeItem{Label: "Bravo", Value: "2", Importance: widget.DangerImportance},
	)
	attributes.Set([]ui.AttributeItem{{Label: "Charlie", Value: "3"}})
	tabs := container.NewAppTabs(
		ui.NewTabItem("Overview", container.NewVBox(
			entity,
			total,
			link,
			ui.NewLabelWithTruncation("A long text which is truncated"),
			ui.NewLabelWithWrapping("A long text which is wrapped"),
			ui.MakeEveEntityActionLabel(nil, func(_ *app.EveEntity) {}),
			ui.MakeCharacterActionLabel(42, "Bruce Wayne", func(_ *app.EveEntity) {}),
			ui.MakeLocationLabel(nil, func(_ int64) {}),
			ui.MakeLinkLabel("Link", func() {}),
			widget.NewLabel(ui.FormatISKAmount(1234567.89)),
		)),
		ui.NewTabItem("Attributes", attributes),
	)
	tabs.Append(container.NewTabItem("Padded", ui.PadTabContent(widget.NewLabel("Padded"))))

	ui.MakeDetailWindow(ui.MakeDetailWindowParams{
		Content:        ui.NewWideScrollBarOverride(tabs),
		EnableTooltips: true,
		ImageLoader: func(setter func(r fyne.Resource)) {
			setter(eis.Character)
		},
		Title:  "Shakedown",
		Window: w,
	})
	w.Resize(fyne.NewSize(800, 600))
	tabs.SelectIndex(1)
	tabs.SelectIndex(2)

	assert.NotNil(t, w.Content())
}
