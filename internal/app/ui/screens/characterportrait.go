package screens

import (
	"context"
	"log/slog"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type CharacterPortrait struct {
	widget.BaseWidget

	character atomic.Pointer[app.Character]
	image     *canvas.Image
	u         baseUI
}

func NewCharacterPortrait(u baseUI) *CharacterPortrait {
	a := &CharacterPortrait{
		image: xwidget.NewImageFromResource(icons.Characterplaceholder512Jpeg, fyne.NewSquareSize(PortraitDesktopUnitSize)),
		u:     u,
	}
	a.ExtendBaseWidget(a)

	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	return a
}

func (a *CharacterPortrait) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(container.NewPadded(a.image))
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterPortrait) update(_ context.Context) {
	id := a.character.Load().IDOrZero()
	if id == 0 {
		fyne.Do(func() {
			a.image.Resource = icons.Characterplaceholder512Jpeg
			a.image.Refresh()
		})
		return
	}
	r, err := a.u.EVEImage().CharacterPortrait(id, PortraitDesktopPixelSize)
	if err != nil {
		slog.Error("Failed to load character portrait", "characterID", id, "error", err)
		fyne.Do(func() {
			a.image.Resource = icons.Characterplaceholder512Jpeg
			a.image.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.image.Resource = r
		a.image.Refresh()
	})
}
