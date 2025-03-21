package character

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ilayout "github.com/ErikKalkoken/evebuddy/internal/layout"
	iwidgets "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	typeIconSize = 55
)

type CharacterAsset struct {
	widget.BaseWidget

	badge      *assetQuantityBadge
	icon       *canvas.Image
	iconLoader func(*canvas.Image, *app.CharacterAsset)
	label      *assetLabel
}

func NewCharacterAsset(iconLoader func(image *canvas.Image, ca *app.CharacterAsset)) *CharacterAsset {
	icon := iwidgets.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(typeIconSize))
	w := &CharacterAsset{
		icon:       icon,
		label:      NewAssetLabel(),
		iconLoader: iconLoader,
		badge:      NewAssetQuantityBadge(),
	}
	w.badge.Hide()
	w.ExtendBaseWidget(w)
	return w
}

func (o *CharacterAsset) Set(ca *app.CharacterAsset) {
	o.label.SetText(ca.DisplayName())
	if !ca.IsSingleton {
		o.badge.SetQuantity(int(ca.Quantity))
		o.badge.Show()
	} else {
		o.badge.Hide()
	}
	o.iconLoader(o.icon, ca)
}

func (o *CharacterAsset) CreateRenderer() fyne.WidgetRenderer {
	customVBox := layout.NewCustomPaddedVBoxLayout(0)
	c := container.NewPadded(container.New(
		customVBox,
		container.New(ilayout.NewBottomRightLayout(), o.icon, o.badge),
		o.label,
	))
	return widget.NewSimpleRenderer(c)
}
