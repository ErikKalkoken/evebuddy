package screens

import (
	"strings"

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
	pinTypeAdvancedProcessor colonyPinType = "Advanced Processor"
	pinTypeBasicProcessor    colonyPinType = "Basic Processor"
	pinTypeCommandCenter     colonyPinType = "Command Center"
	pinTypeExtractor         colonyPinType = "Extractor"
	pinTypeHighTechProcessor colonyPinType = "High-Tech Processor"
	pinTypeSpacePort         colonyPinType = "Launchpad"
	pinTypeStorage           colonyPinType = "Storage"
	pinTypeUnknown           colonyPinType = "???"
)

// iconAndColor returns the icon and its color for a pin type.
func (pt colonyPinType) iconAndColor() (fyne.Resource, fyne.ThemeColorName) {
	switch pt {
	case pinTypeCommandCenter:
		return eveicon.FromName(eveicon.PICommandCenter), theme.ColorNamePrimary
	case pinTypeExtractor:
		return eveicon.FromName(eveicon.PIExtractor), theme.ColorNameSuccess
	case pinTypeBasicProcessor:
		return eveicon.FromName(eveicon.PIProcessor), theme.ColorNameWarning
	case pinTypeAdvancedProcessor:
		return icons.PiprocessoradvancedPng, theme.ColorNameWarning
	case pinTypeHighTechProcessor:
		return icons.PiprocessorhightechPng, theme.ColorNameWarning
	case pinTypeSpacePort:
		return eveicon.FromName(eveicon.PILaunchpad), theme.ColorNamePrimary
	case pinTypeStorage:
		return eveicon.FromName(eveicon.PIStorage), theme.ColorNamePrimary
	}
	return eveicon.FromName(eveicon.Undefined), theme.ColorNameDisabled
}

var installationShortNames = map[string]colonyPinType{
	"Advanced Industry Facility": pinTypeAdvancedProcessor,
	"Basic Industry Facility":    pinTypeBasicProcessor,
	"Command Center":             pinTypeCommandCenter,
	"Extractor Control Unit":     pinTypeExtractor,
	"High-Tech Production Plant": pinTypeHighTechProcessor,
	"Launchpad":                  pinTypeSpacePort,
	"Storage Facility":           pinTypeStorage,
}

// colonyPinTypeOf returns the short type of a pin, e.g. "Extractor".
func colonyPinTypeOf(cp *app.CharacterPlanet, p *app.PlanetPin) colonyPinType {
	n, _ := strings.CutPrefix(p.Type.Name, cp.EvePlanet.TypeDisplay()+" ")
	pinType, ok := installationShortNames[n]
	if !ok {
		return pinTypeUnknown
	}
	return pinType
}

type colonyPinWidget struct {
	widget.BaseWidget

	info   *widget.Label
	name   *widget.Label
	output *widget.Label
	status *xwidget.RichText
	symbol *planetPinSymbol
}

func newColonyPinWidget() *colonyPinWidget {
	status := xwidget.NewRichText()
	name := widget.NewLabel("")
	name.TextStyle.Bold = true
	name.Truncation = fyne.TextTruncateClip
	output := widget.NewLabel("")
	output.Truncation = fyne.TextTruncateClip
	w := &colonyPinWidget{
		info:   widget.NewLabel(""),
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
	w.info.SetText(r.info)
	w.name.SetText(r.name)
	w.output.SetText(r.output)
	w.status.Set(r.status)
	w.symbol.Set(r.symbolIcon, r.symbolIconColor, r.symbolStatusColor, r.progress)
	w.Refresh()
}

var planetPinSymbolCache xsync.Map[string, fyne.Resource]

type planetPinSymbol struct {
	widget.BaseWidget

	icon        fyne.Resource
	iconColor   fyne.ThemeColorName
	progress    optional.Optional[float64] // 0-1, shown as arc
	statusColor fyne.ThemeColorName
}

func newPlanetPinSymbol() *planetPinSymbol {
	w := &planetPinSymbol{
		icon:        icons.BlankSvg,
		iconColor:   theme.ColorNameForeground,
		statusColor: theme.ColorNameDisabled,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *planetPinSymbol) Set(icon fyne.Resource, iconColor fyne.ThemeColorName, statusColor fyne.ThemeColorName, progress optional.Optional[float64]) {
	w.icon = colonyPinIconResource(icon, iconColor)
	w.iconColor = iconColor
	w.progress = progress
	w.statusColor = statusColor
	w.Refresh()
}

func (w *planetPinSymbol) CreateRenderer() fyne.WidgetRenderer {
	c1 := canvas.NewCircle(theme.Color(w.iconColor))               // Outer
	c2 := canvas.NewCircle(theme.Color(theme.ColorNameBackground)) // Middle
	c3 := canvas.NewCircle(theme.Color(theme.ColorNameSeparator))  // Inner

	ic := canvas.NewImageFromResource(w.icon)
	ic.FillMode = canvas.ImageFillContain

	r := &tripleCircleRenderer{
		circles:  []*canvas.Circle{c1, c2, c3},
		icon:     ic,
		progress: canvas.NewArc(0, 0, planetPinSymbolArcCutout, theme.Color(theme.ColorNameForeground)),
		track:    canvas.NewArc(0, 360, planetPinSymbolArcCutout, theme.Color(theme.ColorNameSeparator)),
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
	r.circles[0].FillColor = theme.Color(r.widget.statusColor)
	r.circles[0].Refresh()
	r.track.FillColor = theme.Color(theme.ColorNameSeparator)
	r.track.Refresh()
	if v, ok := r.widget.progress.Value(); ok && v > 0 {
		r.progress.FillColor = theme.Color(theme.ColorNameForeground)
		r.progress.EndAngle = float32(360 * min(v, 1))
		r.progress.Show()
		r.progress.Refresh()
	} else {
		r.progress.Hide()
	}
	r.icon.Resource = r.widget.icon
	r.icon.Refresh()
	canvas.Refresh(r.widget)
}

func (r *tripleCircleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.circles[0], r.circles[1], r.track, r.progress, r.circles[2], r.icon}
}

func (r *tripleCircleRenderer) Destroy() {}

// colonyPinIconResource returns icon tinted in color.
func colonyPinIconResource(icon fyne.Resource, color fyne.ThemeColorName) fyne.Resource {
	key := icon.Name() + string(color)
	if r, ok := planetPinSymbolCache.Load(key); ok {
		return r
	}
	r, err := fynetools.ThemedPNG(icon, theme.Color(color))
	if err != nil {
		fyne.LogError("Failed theme PNG", err)
		return icons.BlankSvg
	}
	planetPinSymbolCache.Store(key, r)
	return r
}
