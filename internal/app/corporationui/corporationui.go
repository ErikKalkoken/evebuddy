package corporationui

import (
	"fmt"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/infowindow"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
)

type UI interface {
	InfoWindow() *infowindow.InfoWindow
	MainWindow() fyne.Window
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
}

type Params struct {
	CharacterService   *characterservice.CharacterService
	CorporationService *corporationservice.CorporationService
	EveImageService    *eveimageservice.EveImageService
	Signals            *app.Signals
	UI                 UI
}

func (arg Params) Services() (services, error) {
	if arg.CharacterService == nil {
		return services{}, fmt.Errorf("corporationui: CharacterService missing")
	}
	if arg.CorporationService == nil {
		return services{}, fmt.Errorf("corporationui: CorporationService missing")
	}
	if arg.EveImageService == nil {
		return services{}, fmt.Errorf("corporationui: EveImageService missing")
	}
	if arg.Signals == nil {
		return services{}, fmt.Errorf("corporationui: Signals missing")
	}
	if arg.UI == nil {
		return services{}, fmt.Errorf("corporationui: UI missing")
	}
	s := services{
		cs:      arg.CharacterService,
		eis:     arg.EveImageService,
		rs:      arg.CorporationService,
		signals: arg.Signals,
		u:       arg.UI,
	}
	return s, nil
}

type services struct {
	cs      *characterservice.CharacterService
	eis     *eveimageservice.EveImageService
	rs      *corporationservice.CorporationService
	signals *app.Signals
	u       UI
}
