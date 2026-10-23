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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xlayout"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// corporationLogoWithLogos is a widget that shows a corporation logo
// together with the logos of the corporation's alliance and faction.
// It has a large version designed for desktop and a small version designed for mobile.
type corporationLogoWithLogos struct {
	widget.BaseWidget

	allianceLogo    *xwidget.TappableImage
	factionLogo     *xwidget.TappableImage
	isSmall         bool
	loadAlliance    loadFuncAsync
	loadCorporation loadFuncAsync
	loadFaction     loadFuncAsync
	logo            *canvas.Image
	logoBackground  *canvas.Rectangle
	logoBorder      *canvas.Rectangle
	showInfo        func(*app.EveEntity)
}

func newCorporationLogoWithLogos(loadCorporation, loadAlliance, loadFaction loadFuncAsync, isSmall bool, showInfo func(*app.EveEntity)) *corporationLogoWithLogos {
	var logo *canvas.Image
	if isSmall {
		logo = xwidget.NewImageFromResource(
			icons.Characterplaceholder256Jpeg,
			fyne.NewSquareSize(PortraitMobileUnitSize),
		)
	} else {
		logo = xwidget.NewImageFromResource(
			icons.Characterplaceholder512Jpeg,
			fyne.NewSquareSize(PortraitDesktopUnitSize),
		)
	}
	w := &corporationLogoWithLogos{
		allianceLogo:    xwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		factionLogo:     xwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		isSmall:         isSmall,
		loadAlliance:    loadAlliance,
		loadCorporation: loadCorporation,
		loadFaction:     loadFaction,
		logo:            logo,
		logoBackground:  canvas.NewRectangle(colorDarkBackground),
		logoBorder:      canvas.NewRectangle(color.Transparent),
		showInfo:        showInfo,
	}
	w.ExtendBaseWidget(w)
	var badgeSize float32
	if isSmall {
		badgeSize = ui.IconUnitSize
	} else {
		badgeSize = 40
	}
	w.allianceLogo.SetFillMode(canvas.ImageFillContain)
	w.allianceLogo.SetMinSize(fyne.NewSquareSize(badgeSize))
	w.allianceLogo.CornerRadius = theme.InputRadiusSize()

	w.factionLogo.SetFillMode(canvas.ImageFillContain)
	w.factionLogo.SetMinSize(fyne.NewSquareSize(badgeSize))
	w.factionLogo.CornerRadius = theme.InputRadiusSize()

	w.logoBackground.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	w.logoBorder.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	w.logoBorder.StrokeWidth = 1
	w.logoBorder.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	return w
}

func (w *corporationLogoWithLogos) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	logo := container.NewStack(w.logoBackground, w.logo, w.logoBorder)
	if w.isSmall {
		c := container.NewVBox(
			logo,
			container.New(layout.NewCustomPaddedLayout(0, -p, 0, 0),
				container.NewHBox(w.allianceLogo, layout.NewSpacer(), w.factionLogo),
			),
		)
		return widget.NewSimpleRenderer(c)
	}
	logoBorder := &layout.CustomPaddedLayout{
		TopPadding:    1 * p,
		BottomPadding: 1 * p,
		LeftPadding:   1 * p,
		RightPadding:  1 * p,
	}
	c := container.NewStack(
		logo,
		container.New(&xlayout.BottomLeftLayout{}, container.New(logoBorder, w.allianceLogo)),
		container.New(&xlayout.BottomRightLayout{}, container.New(logoBorder, w.factionLogo)),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *corporationLogoWithLogos) set(corporationID int64, alliance, faction optional.Optional[*app.EveEntity]) {
	var logoSize int
	if w.isSmall {
		logoSize = PortraitMobilePixelSize
	} else {
		logoSize = PortraitDesktopPixelSize
	}
	w.loadCorporation(corporationID, logoSize, func(r fyne.Resource) {
		w.logo.Resource = r
		w.logo.Refresh()
	})

	const badgeSize = 64

	if alliance, ok := alliance.Value(); ok {
		w.allianceLogo.OnTapped = func() {
			w.showInfo(alliance)
		}
		w.allianceLogo.SetToolTip(alliance.Name)
		w.allianceLogo.Show()
		w.loadAlliance(alliance.ID, badgeSize, func(r fyne.Resource) {
			w.allianceLogo.SetResource(r)
		})
	} else {
		w.allianceLogo.Hide()
	}

	if faction, ok := faction.Value(); ok {
		w.factionLogo.OnTapped = func() {
			w.showInfo(faction)
		}
		w.factionLogo.SetToolTip(faction.Name)
		w.factionLogo.Show()
		w.loadFaction(faction.ID, badgeSize, func(r fyne.Resource) {
			w.factionLogo.SetResource(r)
		})
	} else {
		w.factionLogo.Hide()
	}
}
