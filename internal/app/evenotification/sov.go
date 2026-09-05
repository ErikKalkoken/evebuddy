package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
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
		makeInfoLink(structureType),
		makeInfoLink(solarSystem),
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
		makeInfoLink(corporation),
		makeInfoLink(solarSystem),
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
		makeInfoLink(corporation),
		makeInfoLink(solarSystem),
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
		makeInfoLink(solarSystem),
		makeInfoLink(solarSystem.Constellation),
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
		makeInfoLink(solarSystem),
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
		structureTypeName = makeInfoLink(ee)
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
		makeInfoLink(solarSystem),
		fromLDAPTime(data.DecloakTime).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

func sovDamageRender(ctx context.Context, verb string, aggressorAllianceID, aggressorCorpID, aggressorID, solarSystemID int64, armor, hull, shield float64, eus EVEUniverse) (string, string, error) {
	ids := set.Of(aggressorAllianceID, aggressorCorpID, aggressorID)
	entities, err := eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	solarSystem, err := eus.GetOrCreateSolarSystemESI(ctx, solarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s in %s is under attack", verb, solarSystem.Name)
	body := fmt.Sprintf(
		"The %s in %s is under attack by %s (%s, %s).\n\n"+
			"Armor: **%.1f%%**, Hull: **%.1f%%**, Shield: **%.1f%%**.",
		verb,
		makeInfoLink(solarSystem),
		makeInfoLink(entities[aggressorID]),
		makeInfoLink(entities[aggressorCorpID]),
		makeInfoLink(entities[aggressorAllianceID]),
		armor*100,
		hull*100,
		shield*100,
	)
	return title, body, nil
}

type sovereigntyIHDamageMsg struct {
	baseRenderer
}

func (n sovereigntyIHDamageMsg) entityIDs(text string) (set.Set[int64], error) {
	var data goesi.SovereigntyIHDamageMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return set.Set[int64]{}, err
	}
	return set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID), nil
}

func (n sovereigntyIHDamageMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.SovereigntyIHDamageMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	return sovDamageRender(
		ctx, "Infrastructure Hub",
		data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID, data.SolarSystemID,
		data.ArmorValue, data.HullValue, data.ShieldValue, n.eus,
	)
}

type sovereigntySBUDamageMsg struct {
	baseRenderer
}

func (n sovereigntySBUDamageMsg) entityIDs(text string) (set.Set[int64], error) {
	var data goesi.SovereigntySBUDamageMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return set.Set[int64]{}, err
	}
	return set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID), nil
}

func (n sovereigntySBUDamageMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.SovereigntySBUDamageMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	return sovDamageRender(
		ctx, "Sovereignty Blockade Unit",
		data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID, data.SolarSystemID,
		data.ArmorValue, data.HullValue, data.ShieldValue, n.eus,
	)
}

type sovereigntyTCUDamageMsg struct {
	baseRenderer
}

func (n sovereigntyTCUDamageMsg) entityIDs(text string) (set.Set[int64], error) {
	var data goesi.SovereigntyTCUDamageMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return set.Set[int64]{}, err
	}
	return set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID), nil
}

func (n sovereigntyTCUDamageMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.SovereigntyTCUDamageMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	return sovDamageRender(
		ctx, "Territorial Claim Unit",
		data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID, data.SolarSystemID,
		data.ArmorValue, data.HullValue, data.ShieldValue, n.eus,
	)
}

type sovStationEnteredFreeport struct {
	baseRenderer
}

func (n sovStationEnteredFreeport) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.SovStationEnteredFreeport
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
	title := fmt.Sprintf("%s in %s has entered freeport mode", structureType.Name, solarSystem.Name)
	body := fmt.Sprintf(
		"The %s in %s has entered freeport mode and will exit at **%s**.",
		makeInfoLink(structureType),
		makeInfoLink(solarSystem),
		fromLDAPTime(data.Freeportexittime).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type sovStructureSelfDestructCancel struct {
	baseRenderer
}

func (n sovStructureSelfDestructCancel) unmarshal(text string) (goesi.SovStructureSelfDestructCancel, set.Set[int64], error) {
	var data goesi.SovStructureSelfDestructCancel
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n sovStructureSelfDestructCancel) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n sovStructureSelfDestructCancel) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
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
	title := fmt.Sprintf("Self-destruct of %s in %s cancelled", structureType.Name, solarSystem.Name)
	body := fmt.Sprintf(
		"%s has cancelled the self-destruct of the %s in %s.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(structureType),
		makeInfoLink(solarSystem),
	)
	return title, body, nil
}

type sovStructureSelfDestructFinished struct {
	baseRenderer
}

func (n sovStructureSelfDestructFinished) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.SovStructureSelfDestructFinished
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
	title := fmt.Sprintf("%s in %s has self-destructed", structureType.Name, solarSystem.Name)
	body := fmt.Sprintf(
		"The %s in %s has self-destructed.",
		makeInfoLink(structureType),
		makeInfoLink(solarSystem),
	)
	return title, body, nil
}

type sovStructureSelfDestructRequested struct {
	baseRenderer
}

func (n sovStructureSelfDestructRequested) unmarshal(text string) (goesi.SovStructureSelfDestructRequested, set.Set[int64], error) {
	var data goesi.SovStructureSelfDestructRequested
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n sovStructureSelfDestructRequested) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n sovStructureSelfDestructRequested) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
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
	title := fmt.Sprintf("Self-destruct of %s in %s requested", structureType.Name, solarSystem.Name)
	body := fmt.Sprintf(
		"%s of %s has requested the self-destruct of the %s in %s, "+
			"scheduled for **%s**.",
		makeInfoLink(entities[data.CharID]),
		data.CorpName,
		makeInfoLink(structureType),
		makeInfoLink(solarSystem),
		fromLDAPTime(data.DestructTime).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type allAnchoringMsg struct {
	baseRenderer
}

func (n allAnchoringMsg) unmarshal(text string) (notification2.AllAnchoringMsg, set.Set[int64], error) {
	var data notification2.AllAnchoringMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AllianceID, data.CorpID), nil
}

func (n allAnchoringMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allAnchoringMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	moon, err := n.eus.GetOrCreateMoonESI(ctx, data.MoonID)
	if err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Towers anchoring in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"%s is anchoring **%d** tower(s) at %s in %s.",
		makeInfoLink(entities[data.CorpID]),
		len(data.Towers),
		moon.Name,
		makeInfoLink(solarSystem),
	)
	return title, body, nil
}
