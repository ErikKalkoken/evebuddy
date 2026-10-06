package screens

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type colonyPinType string

const (
	pinTypeAdvancedProcessor colonyPinType = app.PinTypeAdvancedProcessor
	pinTypeBasicProcessor    colonyPinType = app.PinTypeBasicProcessor
	pinTypeCommandCenter     colonyPinType = app.PinTypeCommandCenter
	pinTypeExtractor         colonyPinType = app.PinTypeExtractor
	pinTypeHighTechProcessor colonyPinType = app.PinTypeHighTechProcessor
	pinTypeSpacePort         colonyPinType = app.PinTypeLaunchpad
	pinTypeStorage           colonyPinType = app.PinTypeStorage
	pinTypeUnknown           colonyPinType = "???"
)

// icon returns the icon for a pin type.
func (pt colonyPinType) icon() fyne.Resource {
	switch pt {
	case pinTypeCommandCenter:
		return eveicon.FromName(eveicon.PICommandCenter)
	case pinTypeExtractor:
		return eveicon.FromName(eveicon.PIExtractor)
	case pinTypeBasicProcessor:
		return eveicon.FromName(eveicon.PIProcessor)
	case pinTypeAdvancedProcessor:
		return icons.PiprocessoradvancedPng
	case pinTypeHighTechProcessor:
		return icons.PiprocessorhightechPng
	case pinTypeSpacePort:
		return eveicon.FromName(eveicon.PILaunchpad)
	case pinTypeStorage:
		return eveicon.FromName(eveicon.PIStorage)
	}
	return eveicon.FromName(eveicon.Undefined)
}

// color returns the color of the icon for a pin type when shown in a pin symbol.
func (pt colonyPinType) color() fyne.ThemeColorName {
	switch pt {
	case pinTypeCommandCenter, pinTypeSpacePort, pinTypeStorage:
		return theme.ColorNamePrimary
	case pinTypeExtractor:
		return theme.ColorNameSuccess
	case pinTypeBasicProcessor, pinTypeAdvancedProcessor, pinTypeHighTechProcessor:
		return theme.ColorNameWarning
	}
	return theme.ColorNameDisabled
}

// colonyPinTypeOf returns the short type of a pin, e.g. "Extractor".
func colonyPinTypeOf(cp *app.CharacterPlanet, p *app.PlanetPin) colonyPinType {
	switch pt := colonyPinType(cp.PinTypeName(p)); pt {
	case pinTypeAdvancedProcessor, pinTypeBasicProcessor, pinTypeCommandCenter, pinTypeExtractor,
		pinTypeHighTechProcessor, pinTypeSpacePort, pinTypeStorage:
		return pt
	}
	return pinTypeUnknown
}

type colonyPinWidget struct {
	widget.BaseWidget

	info   *xwidget.RichText
	name   *widget.Label
	output *xwidget.RichText
	status *xwidget.RichText
	symbol *planetPinSymbol
}

func newColonyPinWidget() *colonyPinWidget {
	status := xwidget.NewRichText()
	name := widget.NewLabel("")
	name.TextStyle.Bold = true
	name.Truncation = fyne.TextTruncateEllipsis
	output := xwidget.NewRichText()
	output.Truncation = fyne.TextTruncateClip
	w := &colonyPinWidget{
		info:   xwidget.NewRichText(),
		name:   name,
		output: output,
		status: status,
		symbol: newPlanetPinSymbol(),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *colonyPinWidget) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		container.NewCenter(w.symbol),
		nil,
		container.New(layout.NewCustomPaddedVBoxLayout(-p),
			container.NewBorder(nil, nil, nil, w.status, w.name),
			container.NewBorder(nil, nil, nil, w.info, w.output),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *colonyPinWidget) Set(r colonyDetailsRow) {
	muted := widget.RichTextStyle{ColorName: theme.ColorNamePlaceHolder}
	w.info.SetWithText(r.info, muted)
	w.name.SetText(r.name)
	w.output.SetWithText(r.output, muted)
	w.status.Set(r.status)
	w.symbol.Set(r.symbolIcon, r.symbolIconColor, r.symbolStatusColor, r.progress)
	w.Refresh()
}

type planetPinSymbol struct {
	widget.BaseWidget

	icon        fyne.Resource
	iconColor   fyne.ThemeColorName
	progress    optional.Optional[float64] // 0-1, shown as arc
	statusColor fyne.ThemeColorName
}

func newPlanetPinSymbol() *planetPinSymbol {
	w := &planetPinSymbol{
		iconColor:   theme.ColorNameForeground,
		statusColor: theme.ColorNameDisabled,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *planetPinSymbol) Set(icon fyne.Resource, iconColor fyne.ThemeColorName, statusColor fyne.ThemeColorName, progress optional.Optional[float64]) {
	w.icon = icon
	w.iconColor = iconColor
	w.progress = progress
	w.statusColor = statusColor
	w.Refresh()
}

func (w *planetPinSymbol) CreateRenderer() fyne.WidgetRenderer {
	ic := canvas.NewImageFromResource(nil)
	ic.FillMode = canvas.ImageFillContain
	r := &tripleCircleRenderer{
		circles:  []*canvas.Circle{canvas.NewCircle(nil), canvas.NewCircle(nil), canvas.NewCircle(nil)}, // outer, middle, inner
		icon:     ic,
		progress: canvas.NewArc(0, 0, planetPinSymbolArcCutout, nil),
		track:    canvas.NewArc(0, 360, planetPinSymbolArcCutout, nil),
		widget:   w,
	}
	r.Refresh()
	return r
}

const (
	planetPinSymbolArcCutout = 0.87 // thin ring
	planetPinMinSize         = 50
)

type tripleCircleRenderer struct {
	widget   *planetPinSymbol
	circles  []*canvas.Circle
	icon     *canvas.Image
	progress *canvas.Arc
	tint     iconTint
	track    *canvas.Arc
}

func (r *tripleCircleRenderer) Layout(size fyne.Size) {
	center := fyne.NewPos(size.Width/2, size.Height/2)
	diameter := fyne.Min(size.Width, size.Height)

	diameters := []float32{
		1.0 * diameter,
		0.85 * diameter,
		0.58 * diameter,
	}

	// Layout circles
	for i, circle := range r.circles {
		currentDim := diameters[i]

		circle.Resize(fyne.NewSize(currentDim, currentDim))
		circle.Move(fyne.NewPos(
			center.X-(currentDim/2),
			center.Y-(currentDim/2),
		))
	}

	// Layout the arcs in the gap between the middle and the inner circle.
	// Despite canvas.Arc's doc, its position is the top-left of its bounding box like a circle.
	arcDim := 0.72 * diameter
	for _, arc := range []*canvas.Arc{r.track, r.progress} {
		arc.Resize(fyne.NewSquareSize(arcDim))
		arc.Move(center.Subtract(fyne.NewSquareOffsetPos(arcDim / 2)))
	}

	// Layout the Icon in the center of the smallest circle
	innerCircleDim := diameters[2]
	iconDim := innerCircleDim * 0.7

	r.icon.Resize(fyne.NewSize(iconDim, iconDim))
	r.icon.Move(fyne.NewPos(
		center.X-(iconDim/2),
		center.Y-(iconDim/2),
	))
}

func (r *tripleCircleRenderer) MinSize() fyne.Size {
	return fyne.NewSquareSize(planetPinMinSize)
}

func (r *tripleCircleRenderer) Refresh() {
	th := r.widget.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	for i, name := range []fyne.ThemeColorName{r.widget.statusColor, theme.ColorNameBackground, theme.ColorNameSeparator} {
		r.circles[i].FillColor = th.Color(name, v)
		r.circles[i].Refresh()
	}
	r.track.FillColor = th.Color(theme.ColorNameSeparator, v)
	r.track.Refresh()
	if x, ok := r.widget.progress.Value(); ok && x > 0 {
		r.progress.FillColor = th.Color(theme.ColorNameForeground, v)
		r.progress.EndAngle = float32(360 * min(x, 1))
		r.progress.Show()
		r.progress.Refresh()
	} else {
		r.progress.Hide()
	}
	r.icon.Resource = r.tint.apply(r.widget.icon, th.Color(r.widget.iconColor, v))
	r.icon.Refresh()
}

func (r *tripleCircleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.circles[0], r.circles[1], r.track, r.progress, r.circles[2], r.icon}
}

func (r *tripleCircleRenderer) Destroy() {}

// colonyPinIcon shows the icon of a pin type tinted in the given theme color.
type colonyPinIcon struct {
	widget.BaseWidget

	color   fyne.ThemeColorName
	icon    fyne.Resource
	minSize fyne.Size
}

func newColonyPinIcon(icon fyne.Resource, minSize fyne.Size, color fyne.ThemeColorName) *colonyPinIcon {
	w := &colonyPinIcon{color: color, icon: icon, minSize: minSize}
	w.ExtendBaseWidget(w)
	return w
}

func (w *colonyPinIcon) CreateRenderer() fyne.WidgetRenderer {
	image := canvas.NewImageFromResource(nil)
	image.FillMode = canvas.ImageFillContain
	r := &colonyPinIconRenderer{image: image, widget: w}
	r.Refresh()
	return r
}

type colonyPinIconRenderer struct {
	image  *canvas.Image
	tint   iconTint
	widget *colonyPinIcon
}

func (r *colonyPinIconRenderer) Layout(size fyne.Size) {
	r.image.Resize(size)
}

func (r *colonyPinIconRenderer) MinSize() fyne.Size {
	return r.widget.minSize
}

func (r *colonyPinIconRenderer) Refresh() {
	v := fyne.CurrentApp().Settings().ThemeVariant()
	c := r.widget.Theme().Color(r.widget.color, v)
	r.image.Resource = r.tint.apply(r.widget.icon, c)
	r.image.Refresh()
}

func (r *colonyPinIconRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.image}
}

func (r *colonyPinIconRenderer) Destroy() {}

// iconTint remembers the last tinted icon to skip the cache when nothing changed.
type iconTint struct {
	icon   fyne.Resource
	color  color.NRGBA
	result fyne.Resource
}

func (t *iconTint) apply(icon fyne.Resource, c color.Color) fyne.Resource {
	if icon == nil {
		return nil
	}
	nc := color.NRGBAModel.Convert(c).(color.NRGBA)
	if t.result != nil && t.icon == icon && t.color == nc {
		return t.result
	}
	t.icon, t.color, t.result = icon, nc, colonyTintedIcon(icon, nc)
	return t.result
}

type tintedIconKey struct {
	name  string
	color color.NRGBA
}

var tintedIconCache xsync.Map[tintedIconKey, fyne.Resource]

// colonyTintedIcon returns icon tinted in color.
func colonyTintedIcon(icon fyne.Resource, c color.NRGBA) fyne.Resource {
	key := tintedIconKey{name: icon.Name(), color: c}
	if r, ok := tintedIconCache.Load(key); ok {
		return r
	}
	r, err := fynetools.ThemedPNG(icon, c)
	if err != nil {
		fyne.LogError("Failed theme PNG", err)
		return icons.BlankSvg
	}
	tintedIconCache.Store(key, r)
	return r
}
