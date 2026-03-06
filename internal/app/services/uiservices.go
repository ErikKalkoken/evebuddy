// Package services defines the services using by UI elements.
package services

import (
	"fmt"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/infowindow"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
)

type UI interface {
	InfoWindow() *infowindow.InfoWindow
	MainWindow() fyne.Window
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
}

type Services struct {
	Character   *characterservice.CharacterService
	Corporation *corporationservice.CorporationService
	EVEImage    *eveimageservice.EveImageService
	EVEUniverse *eveuniverseservice.EveUniverseService
	Signals     *app.Signals
	UI          UI
}

func (arg Services) Check() error {
	if arg.Character == nil {
		return fmt.Errorf("overviewui: CharacterService missing")
	}
	if arg.Corporation == nil {
		return fmt.Errorf("overviewui: CorporationService missing")
	}
	if arg.EVEImage == nil {
		return fmt.Errorf("overviewui: EveImageService missing")
	}
	if arg.EVEUniverse == nil {
		return fmt.Errorf("overviewui: EveUniverseService missing")
	}
	if arg.Signals == nil {
		return fmt.Errorf("overviewui: Signals missing")
	}
	if arg.UI == nil {
		return fmt.Errorf("overviewui: UI missing")
	}
	return nil
}
