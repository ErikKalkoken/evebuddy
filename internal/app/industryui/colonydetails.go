package industryui

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

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/app/xtheme"
	"github.com/ErikKalkoken/evebuddy/internal/app/xwindow"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type colonyDetailsRow struct {
	endDate           optional.Optional[string]
	expiryTime        optional.Optional[time.Time]
	groupID           int64
	groupName         string
	name              string
	output            string
	searchTarget      string
	status            []widget.RichTextSegment
	symbolIconColor   fyne.ThemeColorName
	symbolIconName    eveicon.Name
	symbolStatusColor fyne.ThemeColorName
	typeID            int64
}

type colonyDetails struct {
	widget.BaseWidget

	characterID   atomic.Int64
	columnSorter  *xwidget.ColumnSorter[colonyDetailsRow]
	expiryTimes   []time.Time
	footer        *widget.Label
	icon          *canvas.Image
	installations *widget.List
	owner         *widget.Hyperlink
	planet        *xwidget.TappableRichText
	planetID      atomic.Int64
	planetType    *widget.Hyperlink
	region        *widget.Label
	rows          []colonyDetailsRow
	rowsFiltered  []colonyDetailsRow
	search        *widget.Entry
	security      *xwidget.RichText
	selectType2   *kxwidget.FilterChipSelect
	signalKey     string
	sortButton    *xwidget.SortButton
	status        *xwidget.RichText
	u             uiservices.UIServices
}

// showColonyDetailsWindow shows the details of a colony in a window.
func showColonyDetailsWindow(u uiservices.UIServices, r colonyRow) {
	title := fmt.Sprintf("Colony %s", r.planetName)
	key := fmt.Sprintf("colony-%d-%d", r.characterID, r.planetID)
	w, ok, onClosed := u.GetOrCreateWindowWithOnClosed(key, title, r.ownerName)
	if !ok {
		w.Show()
		return
	}

	b := newColonyDetails(u, r.characterID, r.planetID, w)
	err := b.Update(context.Background())
	if err != nil {
		slog.Error(
			"Failed to show colony details",
			slog.Any("characterID", r.characterID),
			slog.Any("planetID", r.planetID),
			slog.Any("error", err),
		)
		xdialog.ShowErrorAndLog("Failed to show colony details", err, u.IsDeveloperMode(), u.MainWindow())
		return
	}

	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		b.stop()
	})

	xwindow.Set(xwindow.Params{
		Content: b,
		Title:   title,
		Window:  w,
		MinSize: fyne.NewSize(600, 700),
	})
	w.Show()
}

func newColonyDetails(u uiservices.UIServices, characterID, planetID int64, w fyne.Window) *colonyDetails {
	if characterID == 0 || planetID == 0 {
		panic(app.ErrInvalid)
	}
	makeHyperLink := func() *widget.Hyperlink {
		x := widget.NewHyperlink("", nil)
		x.Wrapping = fyne.TextWrapWord
		return x
	}
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[colonyDetailsRow]{{
		ID:    1,
		Label: "Group",
		Sort: func(a, b colonyDetailsRow) int {
			return strings.Compare(a.groupName, b.groupName)
		},
	}, {
		ID:    2,
		Label: "Type",
		Sort: func(a, b colonyDetailsRow) int {
			return strings.Compare(a.name, b.name)
		},
	}, {
		ID:    3,
		Label: "End date",
		Sort: func(a, b colonyDetailsRow) int {
			return optional.CompareFunc(a.expiryTime, b.expiryTime, func(a, b time.Time) int {
				return a.Compare(b)
			})
		},
	}}),
		1,
		xwidget.SortAsc,
	)
	planet := xwidget.NewTappableRichText(nil, nil)
	planet.Wrapping = fyne.TextWrapWord
	a := &colonyDetails{
		columnSorter: columnSorter,
		footer:       awidget.NewLabelWithTruncation(""),
		icon:         xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		owner:        makeHyperLink(),
		planet:       planet,
		planetType:   makeHyperLink(),
		region:       widget.NewLabel(""),
		search:       widget.NewEntry(),
		security:     xwidget.NewRichText(),
		signalKey:    u.Signals().UniqueKey(),
		status:       xwidget.NewRichText(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.characterID.Store(characterID)
	a.planetID.Store(planetID)

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
		a.u.InfoWindow().ShowType(a.rowsFiltered[id].typeID)
	}
	a.installations = list

	// filters
	a.selectType2 = kxwidget.NewFilterChipSelect("Type", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync()
	}, w)
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync()
	})
	a.search.OnChanged = func(s string) {
		a.filterRowsAsync()
	}
	a.search.PlaceHolder = "Search"

	// signals
	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.filterRowsAsync()
			a.refreshStatus()
		})
	}, a.signalKey)
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.CharacterID == a.characterID.Load() && arg.Section == app.SectionCharacterPlanets {
			err := a.Update(ctx)
			if err != nil {
				slog.Error("failed to update colony installations", "error", err)
				fyne.Do(func() {
					a.setIssue("ERROR: " + a.u.ErrorDisplay(err))
				})
			}
		}
	}, a.signalKey)
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, o *app.EntityShort) {
		if o.ID == a.characterID.Load() {
			fyne.Do(func() {
				a.setIssue("Character has been removed")
			})
		}
	}, a.signalKey)
	return a
}

func (a *colonyDetails) CreateRenderer() fyne.WidgetRenderer {
	planet := container.NewBorder(nil, nil, a.icon, nil, a.planet)
	infos := widget.NewForm(
		widget.NewFormItem("Planet", planet),
		widget.NewFormItem("Type", a.planetType),
		widget.NewFormItem("Owner", a.owner),
		widget.NewFormItem("Status", a.status),
	)
	// infos.Orientation = widget.Adaptive

	filter := container.NewBorder(
		nil,
		nil,
		container.NewHBox(a.selectType2, a.sortButton),
		nil,
		a.search,
	)

	installations := container.NewBorder(
		container.NewVBox(
			widget.NewSeparator(),
			xwidget.NewStandardSpacer(),
			filter,
		),
		a.footer,
		nil,
		nil,
		a.installations,
	)

	content := container.NewBorder(
		infos,
		nil,
		nil,
		nil,
		installations,
	)
	return widget.NewSimpleRenderer(content)
}

func (a *colonyDetails) stop() {
	a.u.Signals().RefreshTickerExpired.RemoveListener(a.signalKey)
	a.u.Signals().CharacterSectionChanged.RemoveListener(a.signalKey)
	a.u.Signals().CharacterRemoved.RemoveListener(a.signalKey)
}

func (a *colonyDetails) setIssue(s string) {
	a.footer.Text = s
	a.footer.Importance = widget.DangerImportance
	a.footer.Refresh()
}

func (a *colonyDetails) refreshStatus() {
	a.status.Set(colonyStatusDisplay(a.expiryTimes))
}

func (a *colonyDetails) filterRowsAsync() {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	type2 := a.selectType2.Selected
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)

	go func() {
		if type2 != "" {
			rows = slices.DeleteFunc(rows, func(r colonyDetailsRow) bool {
				return r.name != type2
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r colonyDetailsRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}

		typeOptions := xslices.Map(rows, func(r colonyDetailsRow) string {
			return r.name
		})
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		footer := fmt.Sprintf("Showing %d / %d installations", len(rows), totalRows)

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.rowsFiltered = rows
			a.selectType2.SetOptions(typeOptions)
			a.installations.Refresh()

		})
	}()
}

func (a *colonyDetails) Update(ctx context.Context) error {
	cp, rows, err := a.fetchData(ctx)
	if err != nil {
		return err
	}

	ownerName := a.u.StatusCache().CharacterName(a.characterID.Load())
	expiryTimes := cp.ExtractionsExpiryTimes()

	fyne.Do(func() {
		a.u.EVEImage().InventoryTypeIconAsync(cp.EvePlanet.Type.ID, app.IconPixelSize, func(res fyne.Resource) {
			a.icon.Resource = res
			a.icon.Refresh()
		})
		a.security.Set(cp.EvePlanet.SolarSystem.SecurityStatusRichText())
		a.planet.Set(cp.NameRichText())
		a.planet.OnTapped = func() {
			a.u.InfoWindow().Show(app.EveEntitySolarSystem, cp.EvePlanet.SolarSystem.ID)
		}
		a.region.SetText(fmt.Sprintf("(%s)", cp.EvePlanet.SolarSystem.Constellation.Region.Name))
		a.planetType.SetText(cp.EvePlanet.TypeDisplay())
		a.planetType.OnTapped = func() {
			a.u.InfoWindow().ShowEntity(cp.EvePlanet.Type.EveEntity())
		}
		a.owner.SetText(ownerName)
		a.owner.OnTapped = func() {
			a.u.InfoWindow().Show(app.EveEntityCharacter, cp.CharacterID)
		}

		a.expiryTimes = expiryTimes
		a.refreshStatus()

		a.rows = rows
		a.filterRowsAsync()
	})
	return nil
}

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

var installationShortNames = map[string]colonyPinType{
	"Advanced Industry Facility": pinTypeAdvancedProcessor,
	"Basic Industry Facility":    pinTypeBasicProcessor,
	"Command Center":             pinTypeCommandCenter,
	"Extractor Control Unit":     pinTypeExtractor,
	"High-Tech Production Plant": pinTypeHighTechProcessor,
	"Launchpad":                  pinTypeSpacePort,
	"Storage Facility":           pinTypeStorage,
}

func (a *colonyDetails) fetchData(ctx context.Context) (*app.CharacterPlanet, []colonyDetailsRow, error) {
	cp, err := a.u.Character().GetPlanet(ctx, a.characterID.Load(), a.planetID.Load())
	if err != nil {
		return nil, nil, err
	}

	var rows []colonyDetailsRow
	for _, p := range cp.Pins {
		prefix := cp.EvePlanet.TypeDisplay() + " "
		n, _ := strings.CutPrefix(p.Type.Name, prefix)
		pinType, ok := installationShortNames[n]
		if !ok {
			pinType = pinTypeUnknown
		}

		name := string(pinType)
		searchTargets := []string{strings.ToLower(name)}

		var iconColor fyne.ThemeColorName
		var iconName eveicon.Name
		statusColor := theme.ColorNameButton
		switch pinType {
		case pinTypeCommandCenter:
			iconName = eveicon.PICommandCenter
			iconColor = xtheme.ColorNameInfo
		case pinTypeExtractor:
			iconName = eveicon.PIExtractor
			iconColor = xtheme.ColorNameSystem
			if v, ok := p.ExpiryTime.Value(); ok {
				if time.Now().Before(v) {
					statusColor = theme.ColorNameSuccess
				} else {
					statusColor = theme.ColorNameError
				}
			}
		case pinTypeBasicProcessor:
			iconName = eveicon.PIProcessor
			iconColor = theme.ColorNameWarning
		case pinTypeAdvancedProcessor:
			iconName = eveicon.PIProcessor
			iconColor = xtheme.ColorNameAttention
		case pinTypeHighTechProcessor:
			iconName = eveicon.PIProcessor
			iconColor = xtheme.ColorNameCreative
		case pinTypeSpacePort:
			iconName = eveicon.PILaunchpad
			iconColor = theme.ColorNamePrimary
		case pinTypeStorage:
			iconName = eveicon.PIStorage
			iconColor = theme.ColorNamePrimary
		default:
			iconName = eveicon.Undefined
			iconColor = theme.ColorNameDisabled
		}

		var output string
		switch p.Type.Group.ID {
		case app.EveGroupExtractorControlUnits:
			if v, ok := p.ExtractorProductType.Value(); ok {
				output = v.Name
				searchTargets = append(searchTargets, strings.ToLower(v.Name))
			} else {
				output = "-"
			}
		case app.EveGroupProcessors:
			if v, ok := p.Schematic.Value(); ok {
				output = v.Name
				searchTargets = append(searchTargets, strings.ToLower(v.Name))
			} else {
				output = "-"
			}
		case app.EveGroupCommandCenters:
			output = fmt.Sprintf("Level %d", cp.UpgradeLevel)
		}

		var endDate optional.Optional[string]
		var statusText string
		var statusTextColor fyne.ThemeColorName
		if p.Type.Group.ID == app.EveGroupExtractorControlUnits {
			if v, ok := p.ExpiryTime.Value(); !ok {
				statusText = "-"
				statusTextColor = theme.ColorNameDisabled
			} else {
				endDate.Set(v.Format(app.DateTimeFormat))
				if time.Now().After(v) {
					statusText = colonyStatusAllIdle
					statusTextColor = theme.ColorNameError
				} else {
					statusText = ihumanize.Duration(time.Until(v))
					statusTextColor = theme.ColorNameForeground
				}
			}
		}
		status := xwidget.RichTextSegmentsFromText(statusText, widget.RichTextStyle{
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
			searchTarget:      strings.Join(searchTargets, "~"),
		})
	}
	return cp, rows, nil
}

type colonyPinItem struct {
	widget.BaseWidget

	info   *widget.Label
	name   *widget.Label
	output *widget.Label
	status *xwidget.RichText
	symbol *planetPinSymbol
}

func newColonyPinItem() *colonyPinItem {
	status := xwidget.NewRichText()
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
	w.info.SetText(r.endDate.ValueOrZero())
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
