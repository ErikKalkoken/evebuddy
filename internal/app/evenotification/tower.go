package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"github.com/dustin/go-humanize"
	"gopkg.in/yaml.v3"
)

func (s *EveNotificationService) renderTower(ctx context.Context, type_, text string, timestamp time.Time) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case TowerAlertMsg:
		title.Set("Starbase under attack")
		var data notification.TowerAlertMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeTowerBaseText(ctx, data.MoonID, data.TypeID, data.SolarSystemID)
		if err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID})
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf("is under attack.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s",
			makeEveEntityProfileLink(entities[data.AggressorID]),
			makeEveEntityProfileLink(entities[data.AggressorCorpID]),
		)
		if data.AggressorAllianceID != 0 {
			out += fmt.Sprintf(
				"\n\nAttacking Alliance: %s",
				makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
			)
		}
		body.Set(out)

	case TowerResourceAlertMsg:
		title.Set("Starbase fuel alert")
		var data notification.TowerResourceAlertMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeTowerBaseText(ctx, data.MoonID, data.TypeID, data.SolarSystemID)
		if err != nil {
			return title, body, err
		}
		out += "is running out of fuel in less then 24hrs.\n\n"
		if len(data.Wants) > 0 {
			out += fmt.Sprintf(
				"Fuel remaining: %s units", humanize.Comma(int64(data.Wants[0].Quantity)),
			)
		}
		body.Set(out)

	}
	return title, body, nil
}

func (s *EveNotificationService) makeTowerBaseText(ctx context.Context, moonID, typeID, solarSystemID int32) (string, error) {
	structureType, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, typeID)
	if err != nil {
		return "", err
	}
	solarSystem, err := s.EveUniverseService.GetOrCreateEveSolarSystemESI(ctx, solarSystemID)
	if err != nil {
		return "", err
	}
	out := fmt.Sprintf("The %s at %s in %s ", structureType.Name, "???", makeLocationLink(solarSystem))
	return out, nil
}
