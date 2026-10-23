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

// characterPortraitWithLogos is a widget that shows a character portrait
// together with the logos of the character's corporation and alliance.
// It has a large version designed for desktop and a small version designed for mobile.
type characterPortraitWithLogos struct {
	widget.BaseWidget

	allianceLogo    *xwidget.TappableImage
	corporationLogo *xwidget.TappableImage
	isSmall         bool
	loadAlliance    loadFuncAsync
	loadCharacter   loadFuncAsync
	loadCorporation loadFuncAsync
	portrait        *canvas.Image
	portraitBorder  *canvas.Rectangle
	showInfo        func(*app.EveEntity)
}

func newCharacterPortraitWithLogos(loadCharacter, loadCorporation, loadAlliance loadFuncAsync, isSmall bool, showInfo func(*app.EveEntity)) *characterPortraitWithLogos {
	var portrait *canvas.Image
	if isSmall {
		portrait = xwidget.NewImageFromResource(
			icons.Characterplaceholder256Jpeg,
			fyne.NewSquareSize(PortraitMobileUnitSize),
		)
	} else {
		portrait = xwidget.NewImageFromResource(
			icons.Characterplaceholder512Jpeg,
			fyne.NewSquareSize(PortraitDesktopUnitSize),
		)
	}
	w := &characterPortraitWithLogos{
		allianceLogo:    xwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		corporationLogo: xwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		isSmall:         isSmall,
		loadAlliance:    loadAlliance,
		loadCharacter:   loadCharacter,
		loadCorporation: loadCorporation,
		portrait:        portrait,
		portraitBorder:  canvas.NewRectangle(color.Transparent),
		showInfo:        showInfo,
	}
	w.ExtendBaseWidget(w)
	var logoSize float32
	if isSmall {
		logoSize = ui.IconUnitSize
	} else {
		logoSize = 40
	}
	w.allianceLogo.SetFillMode(canvas.ImageFillContain)
	w.allianceLogo.SetMinSize(fyne.NewSquareSize(logoSize))
	w.allianceLogo.CornerRadius = theme.InputRadiusSize()

	w.corporationLogo.SetFillMode(canvas.ImageFillContain)
	w.corporationLogo.SetMinSize(fyne.NewSquareSize(logoSize))
	w.corporationLogo.CornerRadius = theme.InputRadiusSize()

	w.portraitBorder.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	w.portraitBorder.StrokeWidth = 1
	w.portraitBorder.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	return w
}

func (w *characterPortraitWithLogos) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	portrait := container.NewStack(w.portrait, w.portraitBorder)
	if w.isSmall {
		c := container.NewVBox(
			portrait,
			container.New(layout.NewCustomPaddedLayout(0, -p, 0, 0),
				container.NewHBox(
					w.corporationLogo, layout.NewSpacer(), w.allianceLogo,
				),
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
		portrait,
		container.New(&xlayout.BottomLeftLayout{}, container.New(logoBorder, w.corporationLogo)),
		container.New(&xlayout.BottomRightLayout{}, container.New(logoBorder, w.allianceLogo)),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *characterPortraitWithLogos) set(characterID int64, corporation *app.EveEntity, alliance optional.Optional[*app.EveEntity]) {
	var portraitSize int
	if w.isSmall {
		portraitSize = PortraitMobilePixelSize
	} else {
		portraitSize = PortraitDesktopPixelSize
	}
	const logoSize = 64

	w.loadCharacter(characterID, portraitSize, func(r fyne.Resource) {
		w.portrait.Resource = r
		w.portrait.Refresh()
	})

	if !w.isSmall {
		w.corporationLogo.OnTapped = func() {
			w.showInfo(corporation)
		}
		w.corporationLogo.SetToolTip(corporation.Name)
	}

	w.loadCorporation(corporation.ID, logoSize, func(r fyne.Resource) {
		w.corporationLogo.SetResource(r)
	})

	if alliance, ok := alliance.Value(); ok {
		if !w.isSmall {
			w.allianceLogo.OnTapped = func() {
				w.showInfo(alliance)
			}
			w.allianceLogo.SetToolTip(alliance.Name)
		}
		w.allianceLogo.Show()
		w.loadAlliance(alliance.ID, logoSize, func(r fyne.Resource) {
			w.allianceLogo.SetResource(r)
		})
	} else {
		w.allianceLogo.Hide()
	}
}
