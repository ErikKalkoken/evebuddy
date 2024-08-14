package evenotification

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

func (s *EveNotificationService) renderOrbitals(ctx context.Context, type_, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case OrbitalAttacked:
		title.Set("Orbital under attack")
		var data notification.OrbitalAttacked
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeOrbitalBaseText(ctx, data.PlanetID, data.TypeID, data.SolarSystemID)
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

	case OrbitalReinforced:
		title.Set("Orbital reinforced")
		var data notification.OrbitalReinforced
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeOrbitalBaseText(ctx, data.PlanetID, data.TypeID, data.SolarSystemID)
		if err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID})
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf("has been reinforced and will come out at %s.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s",
			fromLDAPTime(data.ReinforceExitTime).Format(app.TimeDefaultFormat),
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
	}
	return title, body, nil
}

func (s *EveNotificationService) makeOrbitalBaseText(ctx context.Context, planetID, typeID, solarSystemID int32) (string, error) {
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
