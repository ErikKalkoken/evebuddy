package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

type colonyDetailsRow struct {
	endDate           string
	expiryTime        optional.Optional[time.Time]
	groupID           int64
	groupName         string
	name              string
	output            string
	status            []widget.RichTextSegment
	symbolIconColor   fyne.ThemeColorName
	symbolIconName    eveicon.Name
	symbolStatusColor fyne.ThemeColorName
	typeID            int64
}

type colonyDetails struct {
	widget.BaseWidget

	characterID   atomic.Int64
	expiryTimes   []time.Time
	icon          *iwidget.TappableImage
	installations *widget.List
	issue         *widget.Label
	owner         *widget.Hyperlink
	planet        *widget.Hyperlink
	planetID      atomic.Int64
	planetType    *widget.Hyperlink
	region        *widget.Label
	rows          []colonyDetailsRow
	rowsFiltered  []colonyDetailsRow
	security      *iwidget.RichText
	signalKey     string
	status        *iwidget.RichText
	u             *baseUI
}

// showColonyDetailsWindow shows the details of a colony in a window.
func showColonyDetailsWindow(u *baseUI, r colonyRow) {
	title := fmt.Sprintf("Colony %s", r.planetName)
	key := fmt.Sprintf("colony-%d-%d", r.characterID, r.planetID)
	w, ok, onClosed := u.getOrCreateWindowWithOnClosed(key, title, r.ownerName)
	if !ok {
		w.Show()
		return
	}

	b := newColonyDetails(u, r.characterID, r.planetID)
	err := b.update(context.Background())
	if err != nil {
		slog.Error(
			"Failed to show colony details",
			slog.Any("characterID", r.characterID),
			slog.Any("planetID", r.planetID),
			slog.Any("error", err),
		)
		u.showErrorDialog("Failed to show colony details", err, u.MainWindow())
		return
	}

	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		b.stop()
	})

	setDetailWindow(detailWindowParams{
		content: b,
		title:   title,
		window:  w,
	})
	w.Show()
}

func newColonyDetails(u *baseUI, characterID, planetID int64) *colonyDetails {
	if characterID == 0 || planetID == 0 {
		panic(app.ErrInvalid)
	}
	makeHyperLink := func() *widget.Hyperlink {
		x := widget.NewHyperlink("?", nil)
		return x
	}
	issue := widget.NewLabel("")
	issue.Wrapping = fyne.TextWrapWord
	issue.Importance = widget.DangerImportance
	issue.Hide()

	a := &colonyDetails{
		issue:      issue,
		signalKey:  fmt.Sprintf("colony-detail-%d-%d-%s", characterID, planetID, uniqueID()),
		status:     iwidget.NewRichText(),
		u:          u,
		security:   iwidget.NewRichText(),
		planet:     makeHyperLink(),
		region:     widget.NewLabel(""),
		owner:      makeHyperLink(),
		planetType: makeHyperLink(),
	}
	a.ExtendBaseWidget(a)

	a.characterID.Store(characterID)
	a.planetID.Store(planetID)

	a.icon = iwidget.NewTappableImage(icons.BlankSvg, nil)
	a.icon.SetFillMode(canvas.ImageFillContain)
	a.icon.SetMinSize(fyne.NewSquareSize(64))

	list := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newColonyPinItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			co.(*colonyPinItem).Set(a.rowsFiltered[id])
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		a.u.ShowTypeInfoWindow(a.rowsFiltered[id].typeID)
	}
	a.installations = list

	// signals
	a.u.refreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.Refresh()
		})
	}, a.signalKey)
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if arg.characterID == a.characterID.Load() && arg.section == app.SectionCharacterPlanets {
			err := a.update(ctx)
			if err != nil {
				slog.Error("failed to update colony installations", "error", err)
				fyne.Do(func() {
					a.setIssue("ERROR: " + a.u.humanizeError(err))
					a.Refresh()
				})
			}
		}
	}, a.signalKey)
	a.u.characterRemoved.AddListener(func(ctx context.Context, o *app.EntityShort) {
		if o.ID == a.characterID.Load() {
			fyne.Do(func() {
				a.setIssue("Character has been removed")
				a.Refresh()
			})
		}
	}, a.signalKey)
	return a
}

func (a *colonyDetails) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	infos := widget.NewForm(
		widget.NewFormItem("Owner", a.owner),
		widget.NewFormItem("Planet", container.New(layout.NewCustomPaddedHBoxLayout(-2*p),
			a.security,
			a.planet,
			a.region,
		)),
		widget.NewFormItem("Type", a.planetType),
		widget.NewFormItem("Status", a.status),
	)
	infos.Orientation = widget.Adaptive

	infosPlus := container.NewBorder(
		a.issue,
		newStandardSpacer(),
		nil,
		container.NewPadded(container.NewVBox((a.icon))),
		infos,
	)

	installations := container.NewBorder(
		widget.NewLabelWithStyle("Installations", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil,
		nil,
		nil,
		a.installations,
	)

	content := container.NewBorder(
		infosPlus,
		nil,
		nil,
		nil,
		installations,
	)
	return widget.NewSimpleRenderer(content)
}

func (a *colonyDetails) stop() {
	a.u.refreshTickerExpired.RemoveListener(a.signalKey)
	a.u.characterSectionChanged.RemoveListener(a.signalKey)
	a.u.characterRemoved.RemoveListener(a.signalKey)
}

func (a *colonyDetails) setIssue(s string) {
	a.issue.SetText(s)
	a.issue.Show()
}

func (a *colonyDetails) Refresh() {
	a.status.Set(colonyStatusDisplay(a.expiryTimes))
	a.installations.Refresh()
	a.BaseWidget.Refresh()
}

func (a *colonyDetails) update(ctx context.Context) error {
	cp, rows, err := a.fetchData(ctx)
	if err != nil {
		return err
	}

	slices.SortFunc(rows, func(a, b colonyDetailsRow) int {
		return strings.Compare(a.groupName, b.groupName)
	})

	ownerName := a.u.scs.CharacterName(a.characterID.Load())
	expiryTimes := cp.ExtractionsExpiryTimes()

	fyne.Do(func() {
		a.issue.Hide()
		a.icon.OnTapped = func() {
			a.u.ShowTypeInfoWindow(cp.EvePlanet.Type.ID)
		}
		a.u.eis.InventoryTypeIconAsync(cp.EvePlanet.Type.ID, app.IconPixelSize, func(res fyne.Resource) {
			a.icon.SetResource(res)
		})
		a.security.Set(cp.EvePlanet.SolarSystem.SecurityStatusRichText())
		a.planet.SetText(cp.EvePlanet.Name)
		a.planet.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntitySolarSystem, cp.EvePlanet.SolarSystem.ID)
		}
		a.region.SetText(fmt.Sprintf("(%s)", cp.EvePlanet.SolarSystem.Constellation.Region.Name))
		a.planetType.SetText(cp.EvePlanet.TypeDisplay())
		a.planetType.OnTapped = func() {
			a.u.ShowEveEntityInfoWindow(cp.EvePlanet.Type.EveEntity())
		}
		a.owner.SetText(ownerName)
		a.owner.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityCharacter, cp.CharacterID)
		}

		a.rows = rows
		a.rowsFiltered = rows
		a.expiryTimes = expiryTimes
		a.Refresh()
	})
	return nil
}

var installationShortNames = map[string]string{
	"Extractor Control Unit":     "Extractor",
	"Basic Industry Facility":    "Basic Processor",
	"Advanced Industry Facility": "Advanced Processor",
	"High-Tech Production Plant": "High-Tech Processor",
	"Storage Facility":           "Storage",
}

func (a *colonyDetails) fetchData(ctx context.Context) (*app.CharacterPlanet, []colonyDetailsRow, error) {
	cp, err := a.u.cs.GetPlanet(ctx, a.characterID.Load(), a.planetID.Load())
	if err != nil {
		return nil, nil, err
	}

	var rows []colonyDetailsRow
	for _, p := range cp.Pins {
		prefix := cp.EvePlanet.TypeDisplay() + " "
		name, _ := strings.CutPrefix(p.Type.Name, prefix)
		if short, ok := installationShortNames[name]; ok {
			name = short
		}

		var iconColor, statusColor fyne.ThemeColorName
		var iconName eveicon.Name
		switch p.Type.Group.ID {
		case app.EveGroupCommandCenters:
			iconName = eveicon.PICommandCenter
			iconColor = colorNameInfo
			statusColor = iconColor
		case app.EveGroupExtractorControlUnits:
			iconName = eveicon.PIExtractor
			iconColor = colorNameSystem
			if v, ok := p.ExpiryTime.Value(); ok && time.Now().After(v) {
				statusColor = theme.ColorNameError
			} else {
				statusColor = iconColor
			}
		case app.EveGroupProcessors:
			iconName = eveicon.PIProcessor
			if strings.Contains(name, "Advanced") {
				iconColor = colorNameAttention
			} else {
				iconColor = theme.ColorNameWarning
			}
			statusColor = iconColor
		case app.EveGroupSpaceports:
			iconName = eveicon.PILaunchpad
			iconColor = theme.ColorNamePrimary
			statusColor = iconColor
		case app.EveGroupStorageFacilities:
			iconName = eveicon.PIStorage
			iconColor = theme.ColorNamePrimary
			statusColor = iconColor
		default:
			iconName = eveicon.Undefined
			iconColor = theme.ColorNameDisabled
			statusColor = iconColor
		}

		var output string
		switch p.Type.Group.ID {
		case app.EveGroupExtractorControlUnits:
			output = p.ExtractorProductType.StringFunc("-", func(v *app.EveType) string {
				return v.Name
			})
		case app.EveGroupProcessors:
			output = p.Schematic.StringFunc("-", func(v *app.EveSchematic) string {
				return v.Name
			})
		case app.EveGroupCommandCenters:
			output = fmt.Sprintf("Level %d", cp.UpgradeLevel)
		}

		var statusText, endDate string
		var statusTextColor fyne.ThemeColorName
		if p.Type.Group.ID == app.EveGroupExtractorControlUnits {
			if v, ok := p.ExpiryTime.Value(); !ok {
				statusText = "-"
				statusTextColor = theme.ColorNameDisabled
			} else {
				endDate = v.Format(app.DateTimeFormat)
				if time.Now().After(v) {
					statusText = colonyStatusAllIdle
					statusTextColor = theme.ColorNameError
				} else {
					statusText = ihumanize.Duration(time.Until(v))
					statusTextColor = theme.ColorNameForeground
				}
			}
		}
		status := iwidget.RichTextSegmentsFromText(statusText, widget.RichTextStyle{
			ColorName: statusTextColor,
		})

		rows = append(rows, colonyDetailsRow{
			endDate:           endDate,
			expiryTime:        p.ExpiryTime,
			groupID:           p.Type.Group.ID,
			groupName:         p.Type.Group.Name,
			name:              name,
			output:            output,
			status:            status,
			symbolIconColor:   iconColor,
			symbolIconName:    iconName,
			symbolStatusColor: statusColor,
			typeID:            p.Type.ID,
		})
	}
	return cp, rows, nil
}

type colonyPinItem struct {
	widget.BaseWidget

	info   *widget.Label
	name   *widget.Label
	output *widget.Label
	status *iwidget.RichText
	symbol *planetPinSymbol
}

func newColonyPinItem() *colonyPinItem {
	status := iwidget.NewRichText()
	name := widget.NewLabel("")
	name.TextStyle.Bold = true
	name.Truncation = fyne.TextTruncateClip
	output := widget.NewLabel("")
	output.Truncation = fyne.TextTruncateClip
	w := &colonyPinItem{
		info:   widget.NewLabel(""),
		name:   name,
		output: output,
		status: status,
		symbol: newPlanetPinSymbol(),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *colonyPinItem) CreateRenderer() fyne.WidgetRenderer {
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

func (w *colonyPinItem) Set(r colonyDetailsRow) {
	w.info.SetText(r.endDate)
	w.name.SetText(r.name)
	w.output.SetText(r.output)
	w.status.Set(r.status)
	w.symbol.Set(eveicon.FromName(r.symbolIconName), r.symbolIconColor, r.symbolStatusColor)
	w.Refresh()
}

var planetPinSymbolCache xsync.Map[string, fyne.Resource]

type planetPinSymbol struct {
	widget.BaseWidget

	icon        fyne.Resource
	iconColor   fyne.ThemeColorName
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

func (w *planetPinSymbol) Set(icon fyne.Resource, iconColor fyne.ThemeColorName, statusColor fyne.ThemeColorName) {
	key := icon.Name() + string(iconColor)
	icon2, ok := planetPinSymbolCache.Load(key)
	if !ok {
		r, err := fynetools.ThemedPNG(icon, theme.Color(iconColor))
		if err != nil {
			fyne.LogError("Failed theme PNG", err)
			icon2 = icons.BlankSvg
		} else {
			planetPinSymbolCache.Store(key, r)
			icon2 = r
		}
	}
	w.icon = icon2
	w.iconColor = iconColor
	w.statusColor = statusColor
	w.Refresh()
}

func (w *planetPinSymbol) CreateRenderer() fyne.WidgetRenderer {
	c1 := canvas.NewCircle(theme.Color(w.iconColor))               // Outer
	c2 := canvas.NewCircle(theme.Color(theme.ColorNameBackground)) // Middle
	c3 := canvas.NewCircle(theme.Color(theme.ColorNameSeparator))  // Inner

	ic := canvas.NewImageFromResource(w.icon)
	ic.FillMode = canvas.ImageFillContain

	return &tripleCircleRenderer{
		circles: []*canvas.Circle{c1, c2, c3},
		icon:    ic,
		widget:  w,
	}
}

type tripleCircleRenderer struct {
	widget  *planetPinSymbol
	circles []*canvas.Circle
	icon    *canvas.Image
}

func (r *tripleCircleRenderer) Layout(size fyne.Size) {
	center := fyne.NewPos(size.Width/2, size.Height/2)
	diameter := fyne.Min(size.Width, size.Height)

	diameters := []float32{
		1.0 * diameter,
		0.85 * diameter,
		0.6 * diameter,
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
	return fyne.NewSquareSize(50)
}

func (r *tripleCircleRenderer) Refresh() {
	r.circles[0].FillColor = theme.Color(r.widget.statusColor)
	r.circles[0].Refresh()
	r.icon.Resource = r.widget.icon
	r.icon.Refresh()
	canvas.Refresh(r.widget)
}

func (r *tripleCircleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.circles[0], r.circles[1], r.circles[2], r.icon}
}

func (r *tripleCircleRenderer) Destroy() {}
