package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

var eventToStructureTypeID = map[int32]int64{
	1: app.EveTypeTCU,
	2: app.EveTypeIHUB,
}

type entosisCaptureStarted struct {
	baseRenderer
}

func (n entosisCaptureStarted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n entosisCaptureStarted) unmarshal(text string) (goesi.EntosisCaptureStarted, set.Set[int64], error) {
	var data goesi.EntosisCaptureStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.StructureTypeID)
	return data, ids, nil
}

func (n entosisCaptureStarted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	structureType, err := n.eus.GetOrCreateEntityESI(ctx, data.StructureTypeID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s in %s is being captured", structureType.Name, solarSystem.Name)
	body = fmt.Sprintf(
		"A capsuleer has started to influence the **%s** in %s with an Entosis Link.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type sovAllClaimAcquiredMsg struct {
	baseRenderer
}

func (n sovAllClaimAcquiredMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovAllClaimAcquiredMsg) unmarshal(text string) (goesi.SovAllClaimAquiredMsg, set.Set[int64], error) {
	var data goesi.SovAllClaimAquiredMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n sovAllClaimAcquiredMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	corporation, err := n.eus.GetOrCreateEntityESI(ctx, data.CorpID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("DED Sovereignty claim acknowledgement: %s", solarSystem.Name)
	body = fmt.Sprintf(
		"This mail is your confirmation that DED now officially acknowledges "+
			"that your member organization %s has claimed sovereignty "+
			"on your behalf in the system %s.",
		makeEveEntityProfileLink(corporation),
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type sovAllClaimLostMsg struct {
	baseRenderer
}

func (n sovAllClaimLostMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovAllClaimLostMsg) unmarshal(text string) (goesi.SovAllClaimLostMsg, set.Set[int64], error) {
	var data goesi.SovAllClaimLostMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n sovAllClaimLostMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	corporation, err := n.eus.GetOrCreateEntityESI(ctx, data.CorpID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Lost sovereignty in: %s", solarSystem.Name)
	body = fmt.Sprintf(
		"DED acknowledges that your member organization %s has lost its claim "+
			"to sovereignty on your behalf in the system %s.",
		makeEveEntityProfileLink(corporation),
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type sovCommandNodeEventStarted struct {
	baseRenderer
}

func (n sovCommandNodeEventStarted) entityIDs(_ string) (set.Set[int64], error) {
	return set.Of[int64](app.EveTypeTCU, app.EveTypeIHUB), nil
}

func (n sovCommandNodeEventStarted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	var data goesi.SovCommandNodeEventStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	structureTypeID, ok := eventToStructureTypeID[data.CampaignEventType]
	var structureTypeName string
	if ok {
		ee, err := n.eus.GetOrCreateEntityESI(ctx, structureTypeID)
		if err != nil {
			return title, body, err
		}
		structureTypeName = ee.Name
	} else {
		structureTypeName = "?"
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"Command nodes for %s in %s have begun to decloak",
		structureTypeName,
		solarSystem.Name,
	)
	body = fmt.Sprintf(
		"Command nodes for %s in %s can now be found throughout the **%s** constellation",
		structureTypeName,
		makeSolarSystemLink(solarSystem),
		solarSystem.Constellation.Name,
	)
	return title, body, nil
}

type sovStructureDestroyed struct {
	baseRenderer
}

func (n sovStructureDestroyed) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovStructureDestroyed) unmarshal(text string) (goesi.SovStructureDestroyed, set.Set[int64], error) {
	var data goesi.SovStructureDestroyed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.StructureTypeID)
	return data, ids, nil
}

func (n sovStructureDestroyed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	structureType, err := n.eus.GetOrCreateEntityESI(ctx, data.StructureTypeID)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s in %s has been destroyed", structureType.Name, solarSystem.Name)
	body = fmt.Sprintf(
		"The command nodes for %s in %s have been destroyed by hostile forces.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type sovStructureReinforced struct {
	baseRenderer
}

func (n sovStructureReinforced) entityIDs(_ string) (set.Set[int64], error) {
	return set.Of[int64](app.EveTypeTCU, app.EveTypeIHUB), nil
}

func (n sovStructureReinforced) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	var data goesi.SovStructureReinforced
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	structureTypeID, ok := eventToStructureTypeID[data.CampaignEventType]
	var structureTypeName string
	if ok {
		ee, err := n.eus.GetOrCreateEntityESI(ctx, structureTypeID)
		if err != nil {
			return title, body, err
		}
		structureTypeName = ee.Name
	} else {
		structureTypeName = "?"
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s in %s has entered reinforced mode", structureTypeName, solarSystem.Name)
	body = fmt.Sprintf(
		"The %s in %s has been reinforced by hostile forces "+
			"and command nodes will begin decloaking at **%s**.",
		structureTypeName,
		makeSolarSystemLink(solarSystem),
		fromLDAPTime(data.DecloakTime).Format(app.DateTimeFormat),
	)
	return title, body, nil
}
