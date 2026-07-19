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
		"A **%s** in %s has entered freeport mode and will exit freeport on **%s**.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
		fromLDAPTime(data.Freeportexittime).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type sovStructureSelfDestructCancel struct {
	baseRenderer
}

func (n sovStructureSelfDestructCancel) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovStructureSelfDestructCancel) unmarshal(text string) (goesi.SovStructureSelfDestructCancel, set.Set[int64], error) {
	var data goesi.SovStructureSelfDestructCancel
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n sovStructureSelfDestructCancel) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	structureType, err := n.eus.GetOrCreateEntityESI(ctx, data.StructureTypeID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Self-destruct of %s in %s cancelled", structureType.Name, solarSystem.Name)
	body = fmt.Sprintf(
		"%s has cancelled the self-destruct sequence for the **%s** in %s.",
		makeEveEntityProfileLink(entities[data.CharID]),
		structureType.Name,
		makeSolarSystemLink(solarSystem),
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
		"The **%s** in %s has completed its self-destruct sequence and been destroyed.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type sovStructureSelfDestructRequested struct {
	baseRenderer
}

func (n sovStructureSelfDestructRequested) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovStructureSelfDestructRequested) unmarshal(text string) (goesi.SovStructureSelfDestructRequested, set.Set[int64], error) {
	var data goesi.SovStructureSelfDestructRequested
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n sovStructureSelfDestructRequested) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	structureType, err := n.eus.GetOrCreateEntityESI(ctx, data.StructureTypeID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Self-destruct requested for %s in %s", structureType.Name, solarSystem.Name)
	body = fmt.Sprintf(
		"%s of **%s** has initiated a self-destruct sequence for the **%s** in %s. "+
			"The structure will be destroyed at **%s**.",
		makeEveEntityProfileLink(entities[data.CharID]),
		data.CorpName,
		structureType.Name,
		makeSolarSystemLink(solarSystem),
		fromLDAPTime(data.DestructTime).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type sovereigntyIHDamageMsg struct {
	baseRenderer
}

func (n sovereigntyIHDamageMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovereigntyIHDamageMsg) unmarshal(text string) (goesi.SovereigntyIHDamageMsg, set.Set[int64], error) {
	var data goesi.SovereigntyIHDamageMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID)
	return data, ids, nil
}

func (n sovereigntyIHDamageMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	title = fmt.Sprintf("Infrastructure Hub in %s under attack", solarSystem.Name)
	body = fmt.Sprintf(
		"The Infrastructure Hub (IHub) in %s is under attack by %s of %s/%s.\n\n"+
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

type sovereigntySBUDamageMsg struct {
	baseRenderer
}

func (n sovereigntySBUDamageMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovereigntySBUDamageMsg) unmarshal(text string) (goesi.SovereigntySBUDamageMsg, set.Set[int64], error) {
	var data goesi.SovereigntySBUDamageMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID)
	return data, ids, nil
}

func (n sovereigntySBUDamageMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	title = fmt.Sprintf("Sovereignty Blockade Unit in %s under attack", solarSystem.Name)
	body = fmt.Sprintf(
		"The Sovereignty Blockade Unit (SBU) in %s is under attack by %s of %s/%s.\n\n"+
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

type sovereigntyTCUDamageMsg struct {
	baseRenderer
}

func (n sovereigntyTCUDamageMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovereigntyTCUDamageMsg) unmarshal(text string) (goesi.SovereigntyTCUDamageMsg, set.Set[int64], error) {
	var data goesi.SovereigntyTCUDamageMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID)
	return data, ids, nil
}

func (n sovereigntyTCUDamageMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	title = fmt.Sprintf("Territorial Claim Unit in %s under attack", solarSystem.Name)
	body = fmt.Sprintf(
		"The Territorial Claim Unit (TCU) in %s is under attack by %s of %s/%s.\n\n"+
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

type sovCorpBillLateMsg struct {
	baseRenderer
}

func (n sovCorpBillLateMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovCorpBillLateMsg) unmarshal(text string) (notification2.SovCorpBillLateMsg, set.Set[int64], error) {
	var data notification2.SovCorpBillLateMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n sovCorpBillLateMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	title = fmt.Sprintf("Sovereignty bill late in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"The sovereignty bill for %s in %s is late. "+
			"Sovereignty will be lost if the bill is not paid.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type sovCorpClaimFailMsg struct {
	baseRenderer
}

func (n sovCorpClaimFailMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovCorpClaimFailMsg) unmarshal(text string) (notification2.SovCorpClaimFailMsg, set.Set[int64], error) {
	var data notification2.SovCorpClaimFailMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n sovCorpClaimFailMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	title = fmt.Sprintf("Sovereignty claim failed for %s in %s", entities[data.CorpID].Name, solarSystem.Name)
	body = fmt.Sprintf(
		"The sovereignty claim by %s in %s has failed.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type sovDisruptorMsg struct {
	baseRenderer
}

func (n sovDisruptorMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n sovDisruptorMsg) unmarshal(text string) (notification2.SovDisruptorMsg, set.Set[int64], error) {
	var data notification2.SovDisruptorMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorID)
	return data, ids, nil
}

func (n sovDisruptorMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	title = fmt.Sprintf("Sovereignty disrupted in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"%s has disrupted sovereignty in %s.",
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}
