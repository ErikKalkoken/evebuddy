package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
)

type skyhookDeployed struct {
	baseRenderer
}

func (n skyhookDeployed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.SkyhookDeployed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	planet, err := n.eus.GetOrCreatePlanetESI(ctx, data.PlanetID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Skyhook deployed in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"A Skyhook has been deployed in %s near %s.",
		makeSolarSystemLink(solarSystem),
		planet.Name,
	)
	return title, body, nil
}

type skyhookDestroyed struct {
	baseRenderer
}

func (n skyhookDestroyed) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n skyhookDestroyed) unmarshal(text string) (notification2.SkyhookDestroyed, set.Set[int64], error) {
	var data notification2.SkyhookDestroyed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorAllianceID, data.AggressorCharacterID, data.AggressorCorpID)
	return data, ids, nil
}

func (n skyhookDestroyed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	planet, err := n.eus.GetOrCreatePlanetESI(ctx, data.PlanetID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Skyhook destroyed in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"Your Skyhook near %s in %s has been destroyed by %s of %s/%s.",
		planet.Name,
		makeSolarSystemLink(solarSystem),
		makeEveEntityProfileLink(entities[data.AggressorCharacterID]),
		makeEveEntityProfileLink(entities[data.AggressorCorpID]),
		makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
	)
	return title, body, nil
}

type skyhookLostShields struct {
	baseRenderer
}

func (n skyhookLostShields) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.SkyhookLostShields
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarsystemID)
	if err != nil {
		return "", "", err
	}
	planet, err := n.eus.GetOrCreatePlanetESI(ctx, data.PlanetID)
	if err != nil {
		return "", "", err
	}
	timeLeft := fromLDAPDuration(data.TimeLeft)
	vulnerableTime := fromLDAPDuration(data.VulnerableTime)
	title := fmt.Sprintf("Skyhook lost shields in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"Your Skyhook near %s in %s has lost its shields and is now in reinforcement.\n\n"+
			"The structure will be vulnerable for **%s** after a reinforcement period of **%s**.",
		planet.Name,
		makeSolarSystemLink(solarSystem),
		vulnerableTime.String(),
		timeLeft.String(),
	)
	return title, body, nil
}

type skyhookOnline struct {
	baseRenderer
}

func (n skyhookOnline) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.SkyhookOnline
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	planet, err := n.eus.GetOrCreatePlanetESI(ctx, data.PlanetID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Skyhook online in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"Your Skyhook near %s in %s has come online.",
		planet.Name,
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type skyhookUnderAttack struct {
	baseRenderer
}

func (n skyhookUnderAttack) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n skyhookUnderAttack) unmarshal(text string) (goesi.SkyhookUnderAttack, set.Set[int64], error) {
	var data goesi.SkyhookUnderAttack
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllianceID, data.CharID)
	return data, ids, nil
}

func (n skyhookUnderAttack) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarsystemID)
	if err != nil {
		return title, body, err
	}
	planet, err := n.eus.GetOrCreatePlanetESI(ctx, data.PlanetID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Skyhook under attack in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"Your Skyhook near %s in %s is under attack by %s of **%s**/%s.\n\n"+
			"Shield: **%.1f%%** Armor: **%.1f%%** Hull: **%.1f%%**",
		planet.Name,
		makeSolarSystemLink(solarSystem),
		makeEveEntityProfileLink(entities[data.CharID]),
		data.CorpName,
		makeEveEntityProfileLink(entities[data.AllianceID]),
		data.ShieldPercentage,
		data.ArmorPercentage,
		data.HullPercentage,
	)
	return title, body, nil
}

type stationServiceDisabled struct {
	baseRenderer
}

func (n stationServiceDisabled) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.StationServiceDisabled
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	structureType, err := n.eus.GetOrCreateEntityESI(ctx, data.StructureTypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s disabled in %s", structureType.Name, solarSystem.Name)
	body := fmt.Sprintf(
		"The **%s** service has been disabled in %s.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type stationServiceEnabled struct {
	baseRenderer
}

func (n stationServiceEnabled) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.StationServiceEnabled
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	structureType, err := n.eus.GetOrCreateEntityESI(ctx, data.StructureTypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s enabled in %s", structureType.Name, solarSystem.Name)
	body := fmt.Sprintf(
		"The **%s** service has been enabled in %s.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type stationAggressionMsg struct {
	baseRenderer
}

func (n stationAggressionMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n stationAggressionMsg) unmarshal(text string) (notification2.StationAggressionMsg, set.Set[int64], error) {
	var data notification2.StationAggressionMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID)
	return data, ids, nil
}

func (n stationAggressionMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Station under attack in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"A station in %s is under attack by %s of %s/%s.\n\n"+
			"Shield: **%.0f%%** Armor: **%.0f%%** Hull: **%.0f%%**",
		makeSolarSystemLink(solarSystem),
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.AggressorCorpID]),
		makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
		data.ShieldValue*100,
		data.ArmorValue*100,
		data.HullValue*100,
	)
	return title, body, nil
}

type stationConquerMsg struct {
	baseRenderer
}

func (n stationConquerMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n stationConquerMsg) unmarshal(text string) (notification2.StationConquerMsg, set.Set[int64], error) {
	var data notification2.StationConquerMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.NewOwnerCorpID, data.OldOwnerCorpID)
	return data, ids, nil
}

func (n stationConquerMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Station conquered in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"A station in %s has been conquered by %s from %s.",
		makeSolarSystemLink(solarSystem),
		makeEveEntityProfileLink(entities[data.NewOwnerCorpID]),
		makeEveEntityProfileLink(entities[data.OldOwnerCorpID]),
	)
	return title, body, nil
}

type stationStateChangeMsg struct {
	baseRenderer
}

func (n stationStateChangeMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.StationStateChangeMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	state := data.State
	if state == "" {
		state = "unknown"
	}
	title := fmt.Sprintf("Station state changed in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"A station in %s has changed state to **%s**.",
		makeSolarSystemLink(solarSystem),
		state,
	)
	return title, body, nil
}
