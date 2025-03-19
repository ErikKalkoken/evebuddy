package evenotification

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

func (s *EveNotificationService) renderOrbital(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case OrbitalAttacked:
		var data notification.OrbitalAttacked
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeOrbitalBaseText(ctx, data.PlanetID, data.TypeID)
		if err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s at %s is under attack",
			o.type_.Name,
			o.planet.Name,
		))
		t := fmt.Sprintf("%s is under attack.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s",
			o.intro,
			makeEveEntityProfileLink(entities[data.AggressorID]),
			makeEveEntityProfileLink(entities[data.AggressorCorpID]),
		)
		if data.AggressorAllianceID != 0 {
			t += fmt.Sprintf(
				"\n\nAttacking Alliance: %s",
				makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
			)
		}
		body.Set(t)

	case OrbitalReinforced:
		var data notification.OrbitalReinforced
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeOrbitalBaseText(ctx, data.PlanetID, data.TypeID)
		if err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s at %s has been reinforced",
			o.type_.Name,
			o.planet.Name,
		))
		t := fmt.Sprintf("has been reinforced and will come out at %s.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s",
			fromLDAPTime(data.ReinforceExitTime).Format(app.DateTimeFormat),
			makeEveEntityProfileLink(entities[data.AggressorID]),
			makeEveEntityProfileLink(entities[data.AggressorCorpID]),
		)
		if data.AggressorAllianceID != 0 {
			t += fmt.Sprintf(
				"\n\nAttacking Alliance: %s",
				makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
			)
		}
		body.Set(t)
	}
	return title, body, nil
}

type orbitalInfo struct {
	type_  *app.EveType
	planet *app.EvePlanet
	intro  string
}

func (s *EveNotificationService) makeOrbitalBaseText(ctx context.Context, planetID, typeID int32) (orbitalInfo, error) {
	structureType, err := s.EveUniverseService.GetOrCreateTypeESI(ctx, typeID)
	if err != nil {
		return orbitalInfo{}, err
	}
	planet, err := s.EveUniverseService.GetOrCreatePlanetESI(ctx, planetID)
	if err != nil {
		return orbitalInfo{}, err
	}
	into := fmt.Sprintf(
		"The %s at %s in %s ",
		structureType.Name,
		planet.Name,
		makeSolarSystemLink(planet.SolarSystem),
	)
	x := orbitalInfo{
		type_:  structureType,
		planet: planet,
		intro:  into,
	}
	return x, nil
}
