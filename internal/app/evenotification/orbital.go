package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

type orbitalAttacked struct {
	baseRenderer
}

func (n orbitalAttacked) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n orbitalAttacked) unmarshal(text string) (notification.OrbitalAttacked, setInt32, error) {
	var data notification.OrbitalAttacked
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID)
	return data, ids, nil
}

func (n orbitalAttacked) render(ctx context.Context, text string, timestamp time.Time) (optionalString, optionalString, error) {
	var title, body optionalString
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	o, err := makeOrbitalBaseText(ctx, data.PlanetID, data.TypeID, n.eus)
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
	return title, body, nil
}

type orbitalReinforced struct {
	baseRenderer
}

func (n orbitalReinforced) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n orbitalReinforced) unmarshal(text string) (notification.OrbitalReinforced, setInt32, error) {
	var data notification.OrbitalReinforced
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID)
	return data, ids, nil
}

func (n orbitalReinforced) render(ctx context.Context, text string, timestamp time.Time) (optionalString, optionalString, error) {
	var title, body optionalString
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	o, err := makeOrbitalBaseText(ctx, data.PlanetID, data.TypeID, n.eus)
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
	return title, body, nil
}

type orbitalInfo struct {
	type_  *app.EveType
	planet *app.EvePlanet
	intro  string
}

func makeOrbitalBaseText(ctx context.Context, planetID, typeID int32, eus *eveuniverseservice.EveUniverseService) (orbitalInfo, error) {
	structureType, err := eus.GetOrCreateTypeESI(ctx, typeID)
	if err != nil {
		return orbitalInfo{}, err
	}
	planet, err := eus.GetOrCreatePlanetESI(ctx, planetID)
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
