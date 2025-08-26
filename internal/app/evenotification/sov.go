package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/notification"
)

var eventToStructureTypeID = map[int32]int32{
	1: app.EveTypeTCU,
	2: app.EveTypeIHUB,
}

type entosisCaptureStarted struct {
	baseRenderer
}

func (n entosisCaptureStarted) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n entosisCaptureStarted) unmarshal(text string) (notification.EntosisCaptureStarted, setInt32, error) {
	var data notification.EntosisCaptureStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.StructureTypeID)
	return data, ids, nil
}

func (n entosisCaptureStarted) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n sovAllClaimAcquiredMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n sovAllClaimAcquiredMsg) unmarshal(text string) (notification.SovAllClaimAquiredMsg, setInt32, error) {
	var data notification.SovAllClaimAquiredMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n sovAllClaimAcquiredMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n sovAllClaimLostMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n sovAllClaimLostMsg) unmarshal(text string) (notification.SovAllClaimLostMsg, setInt32, error) {
	var data notification.SovAllClaimLostMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n sovAllClaimLostMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n sovCommandNodeEventStarted) entityIDs(text string) (setInt32, error) {
	return set.Of[int32](app.EveTypeTCU, app.EveTypeIHUB), nil
}

func (n sovCommandNodeEventStarted) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.SovCommandNodeEventStarted
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

func (n sovStructureDestroyed) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n sovStructureDestroyed) unmarshal(text string) (notification.SovStructureDestroyed, setInt32, error) {
	var data notification.SovStructureDestroyed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.StructureTypeID)
	return data, ids, nil
}

func (n sovStructureDestroyed) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n sovStructureReinforced) entityIDs(text string) (setInt32, error) {
	return set.Of[int32](app.EveTypeTCU, app.EveTypeIHUB), nil
}

func (n sovStructureReinforced) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.SovStructureReinforced
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
