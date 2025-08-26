package evenotification

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/antihax/goesi/notification"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type eveObj struct {
	ID   int
	Name string
}

type structureInfo struct {
	eveType     eveObj
	intro       string
	name        string
	owner       eveObj
	solarSystem eveObj
}

func makeStructureBaseText(ctx context.Context, typeID, systemID int32, structureID int64, structureName string, eus *eveuniverseservice.EveUniverseService) (structureInfo, error) {
	var eveType *app.EveType
	var err error
	if typeID != 0 {
		eveType, err = eus.GetOrCreateTypeESI(ctx, typeID)
		if err != nil {
			return structureInfo{}, err
		}
	}
	system, err := eus.GetOrCreateSolarSystemESI(ctx, systemID)
	if err != nil {
		return structureInfo{}, err
	}
	var ownerLink string
	var owner *app.EveEntity
	isUpwellStructure := eveType != nil && eveType.Group.Category.ID == app.EveCategoryStructure
	if isUpwellStructure {
		structure, err := eus.GetOrCreateLocationESI(ctx, structureID)
		if err != nil {
			return structureInfo{}, err
		}
		if structure.Variant() == app.EveLocationStructure {
			structureName = structure.DisplayName2()
			if structure.Owner != nil {
				owner = structure.Owner
				ownerLink = makeEveEntityProfileLink(structure.Owner)
			}
		}
	}
	var name string
	if eveType != nil {
		isOrbital := eveType.Group.Category.ID == app.EveCategoryOrbitals
		if isOrbital && structureName != "" {
			name = fmt.Sprintf("**%s**", structureName)
		} else if structureName != "" {
			name = fmt.Sprintf("%s **%s**", eveType.Name, structureName)
		} else {
			name = eveType.Name
		}
	} else if structureName != "" {
		name = structureName
	} else {
		name = "unknown structure"
	}
	text := fmt.Sprintf("The %s in %s", name, makeSolarSystemLink(system))
	if ownerLink != "" {
		text += fmt.Sprintf(" belonging to %s", ownerLink)
	}
	x := structureInfo{
		solarSystem: eveObj{ID: int(system.ID), Name: system.Name},
		name:        structureName,
		intro:       text,
	}
	if eveType != nil {
		x.eveType.ID = int(eveType.ID)
		x.eveType.Name = eveType.Name
	}
	if owner != nil {
		x.owner.ID = int(owner.ID)
		x.owner.Name = owner.Name
	}
	return x, nil
}

type ownershipTransferred struct {
	baseRenderer
}

func (n ownershipTransferred) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n ownershipTransferred) unmarshal(text string) (notification.OwnershipTransferredV2, setInt32, error) {
	var data notification.OwnershipTransferredV2
	if strings.Contains(text, "newOwnerCorpID") {
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return data, setInt32{}, err
		}
	} else {
		var data2 notification.OwnershipTransferred
		if err := yaml.Unmarshal([]byte(text), &data2); err != nil {
			return data, setInt32{}, err
		}
		data.CharID = int32(data2.CharacterLinkData[2].(uint64))
		data.NewOwnerCorpID = int32(data2.ToCorporationLinkData[2].(uint64))
		data.OldOwnerCorpID = int32(data2.FromCorporationLinkData[2].(uint64))
		data.SolarSystemID = int32(data2.SolarSystemLinkData[2].(uint64))
		data.StructureID = int64(data2.StructureLinkData[2].(uint64))
		data.StructureTypeID = int32(data2.StructureLinkData[1].(uint64))
		data.StructureName = data2.StructureName
	}
	ids := set.Of(data.OldOwnerCorpID, data.NewOwnerCorpID, data.CharID)
	return data, ids, nil
}

func (n ownershipTransferred) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	d, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, d.StructureTypeID, d.SolarSystemID, d.StructureID, d.StructureName, n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s ownership has been transferred to %s",
		d.StructureName,
		entities[d.NewOwnerCorpID].Name,
	)
	body = fmt.Sprintf(
		"%s has been transferred from %s to %s by %s.",
		o.intro,
		makeEveEntityProfileLink(entities[d.OldOwnerCorpID]),
		makeEveEntityProfileLink(entities[d.NewOwnerCorpID]),
		makeEveEntityProfileLink(entities[d.CharID]),
	)
	return title, body, nil
}

type structureAnchoring struct {
	baseRenderer
}

func (n structureAnchoring) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureAnchoring
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"A %s has started anchoring in %s",
		o.eveType.Name,
		o.solarSystem.Name,
	)
	body = fmt.Sprintf("%s has started anchoring.", o.intro)
	return title, body, nil
}

type structureDestroyed struct {
	baseRenderer
}

func (n structureDestroyed) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureDestroyed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s in %s has been destroyed",
		o.name,
		o.solarSystem.Name,
	)
	body = fmt.Sprintf(
		"%s has been destroyed. Item located inside the structure are available for transfer to asset safety.",
		o.intro,
	)
	return title, body, nil
}

type structureFuelAlert struct {
	baseRenderer
}

func (n structureFuelAlert) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureFuelAlert
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s in %s is low on fuel",
		o.name,
		o.solarSystem.Name,
	)
	body = fmt.Sprintf("%s is running out of fuel in 24hrs.", o.intro)
	return title, body, nil
}

type structureImpendingAbandonmentAssetsAtRisk struct {
	baseRenderer
}

func (n structureImpendingAbandonmentAssetsAtRisk) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureImpendingAbandonmentAssetsAtRisk
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	name := evehtml.Strip(data.StructureLink)
	title = fmt.Sprintf("Your assets located in %s are at risk", name)
	body = fmt.Sprintf(
		"You have assets located at **%s** in %s. "+
			"These assets are at risk of loss as the structure is close to becoming abandoned.\n\n"+
			"In approximately %d days this structure will become abandoned.",
		name,
		makeSolarSystemLink(solarSystem),
		data.DaysUntilAbandon,
	)
	return title, body, nil
}

type structureItemsDelivered struct {
	baseRenderer
}

func (n structureItemsDelivered) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n structureItemsDelivered) unmarshal(text string) (notification.StructureItemsDelivered, setInt32, error) {
	var data notification.StructureItemsDelivered
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID, data.StructureTypeID)
	for _, r := range data.ListOfTypesAndQty {
		ids.Add(r[1])
	}
	return data, ids, nil
}

func (n structureItemsDelivered) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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
	structure, err := n.eus.GetOrCreateLocationESI(ctx, data.StructureID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Items delivered from %s", entities[data.CharID].Name)
	var location string
	if structure.Name != "" {
		location = fmt.Sprintf("**%s**", structure.Name)
	} else {
		location = fmt.Sprintf("a %s", makeEveEntityProfileLink(entities[data.StructureTypeID]))
	}
	b := fmt.Sprintf(
		"%s has delivered the following items to %s in %s:\n\n",
		makeEveEntityProfileLink(entities[data.CharID]),
		location,
		makeSolarSystemLink(solarSystem),
	)
	for _, r := range data.ListOfTypesAndQty {
		b += fmt.Sprintf("%dx %s\n\n", r[0], entities[r[1]].Name)
	}
	body = b
	return title, body, nil
}

type structureItemsMovedToSafety struct {
	baseRenderer
}

func (n structureItemsMovedToSafety) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n structureItemsMovedToSafety) unmarshal(text string) (notification.StructureItemsMovedToSafety, setInt32, error) {
	var data notification.StructureItemsMovedToSafety
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.NewStationID)
	return data, ids, nil
}

func (n structureItemsMovedToSafety) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	station, err := n.eus.GetOrCreateEntityESI(ctx, data.NewStationID)
	if err != nil {
		return title, body, err
	}
	name := evehtml.Strip(data.StructureLink)
	title = fmt.Sprintf("Your assets located in %s have been moved to asset safety", name)
	body = fmt.Sprintf(
		"You assets located at **%s** in %s have been moved to asset safety.\n\n"+
			"They can be moved to a location of your choosing earliest at %s.\n\n"+
			"They will be moved automatically to %s by %s.",
		name,
		makeSolarSystemLink(solarSystem),
		fromLDAPTime(data.AssetSafetyMinimumTimestamp).Format(app.DateTimeFormat),
		station.Name,
		fromLDAPTime(data.AssetSafetyFullTimestamp).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type structureLostArmor struct {
	baseRenderer
}

func (n structureLostArmor) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureLostArmor
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s in %s has lost it's armor",
		o.name,
		o.solarSystem.Name,
	)
	body = fmt.Sprintf(
		"%s has lost it's armor. Hull timer ends at **%s**.",
		o.intro,
		fromLDAPTime(data.Timestamp).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type structureLostShields struct {
	baseRenderer
}

func (n structureLostShields) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureLostShields
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s in %s has lost it's shields",
		o.name,
		o.solarSystem.Name,
	)
	body = fmt.Sprintf(
		"%s has lost it's shields and is now in reinforcement state. "+
			"It will exit reinforcement at **%s** and will then be vulnerable for 15 minutes.",
		o.intro,
		fromLDAPTime(data.Timestamp).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type structureOnline struct {
	baseRenderer
}

func (n structureOnline) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureOnline
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s in %s is now online",
		o.name,
		o.solarSystem.Name,
	)
	body = fmt.Sprintf("%s is now online.", o.intro)
	return title, body, nil
}

type structuresReinforcementChanged struct {
	baseRenderer
}

func (n structuresReinforcementChanged) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n structuresReinforcementChanged) unmarshal(text string) (notification.StructuresReinforcementChanged, setInt32, error) {
	var data notification.StructuresReinforcementChanged
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	var ids setInt32
	for _, r := range data.AllStructureInfo {
		ids.Add(int32(r[2].(uint64)))
	}
	return data, ids, nil
}

type structureReinforcementInfo struct {
	structureID int64
	name        string
	typeID      int32
}

func (n structuresReinforcementChanged) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	data, typeIDs, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	structures := make([]structureReinforcementInfo, 0)
	for _, r := range data.AllStructureInfo {
		typeID := int32(r[2].(uint64))
		s := structureReinforcementInfo{
			structureID: int64(r[0].(uint64)),
			name:        r[1].(string),
			typeID:      typeID,
		}
		structures = append(structures, s)
	}
	slices.SortFunc(structures, func(a structureReinforcementInfo, b structureReinforcementInfo) int {
		return cmp.Compare(a.name, b.name)
	})
	entities, err := n.eus.ToEntities(ctx, typeIDs)
	if err != nil {
		return title, body, err
	}
	lines := make([]string, 0)
	for _, o := range structures {
		lines = append(lines, fmt.Sprintf("- %s (%s)", o.name, entities[o.typeID].Name))
	}
	title = "Structure reinforcement time changed"
	out := fmt.Sprintf(
		"Reinforcement hour has been changed to %d:00 "+
			"for the following structures:\n\n%s",
		data.Hour,
		strings.Join(lines, "\n\n"),
	)
	body = out
	return title, body, nil
}

type structureServicesOffline struct {
	baseRenderer
}

func (n structureServicesOffline) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n structureServicesOffline) unmarshal(text string) (notification.StructureServicesOffline, setInt32, error) {
	var data notification.StructureServicesOffline
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.ListOfServiceModuleIDs...)
	return data, ids, nil
}

func (n structureServicesOffline) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	lines := make([]string, 0)
	for e := range maps.Values(entities) {
		lines = append(lines, fmt.Sprintf("- %s", e.Name))
	}
	slices.Sort(lines)
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s in %s has all services off-lined",
		o.name,
		o.solarSystem.Name,
	)
	body = fmt.Sprintf(
		"%s has all services off-lined.\n\n%s",
		o.intro,
		strings.Join(lines, "\n\n"),
	)
	return title, body, nil
}

type structureUnanchoring struct {
	baseRenderer
}

func (n structureUnanchoring) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureUnanchoring
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s has started unanchoring in %s",
		o.name,
		o.solarSystem.Name,
	)
	due := timestamp.Add(fromLDAPDuration(data.TimeLeft))
	body = fmt.Sprintf(
		"%s has started un-anchoring. It will be fully un-anchored at: %s",
		o.intro,
		due.Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type structureUnderAttack struct {
	baseRenderer
}

func (n structureUnderAttack) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n structureUnderAttack) unmarshal(text string) (notification.StructureUnderAttack, setInt32, error) {
	var data notification.StructureUnderAttack
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n structureUnderAttack) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s in %s is under attack",
		o.name,
		o.solarSystem.Name,
	)
	attackChar, err := n.eus.GetOrCreateEntityESI(ctx, data.CharID)
	if err != nil {
		return title, body, err
	}
	t := fmt.Sprintf("%s is under attack.\n\n"+
		"Attacking Character: %s\n\n"+
		"Attacking Corporation: %s",
		o.intro,
		makeEveEntityProfileLink(attackChar),
		makeCorporationLink(data.CorpName),
	)
	if data.AllianceName != "" {
		t += fmt.Sprintf(
			"\n\nAttacking Alliance: %s",
			makeAllianceLink(data.AllianceName),
		)
	}
	body = t
	return title, body, nil
}

type structureWentHighPower struct {
	baseRenderer
}

func (n structureWentHighPower) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureWentHighPower
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s is now running on High Power", o.name)
	body = fmt.Sprintf("%s went to high power mode.", o.intro)
	return title, body, nil
}

type structureWentLowPower struct {
	baseRenderer
}

func (n structureWentLowPower) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.StructureWentLowPower
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "", n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s is now running on Low Power", o.name)
	body = fmt.Sprintf("%s went to low power mode.", o.intro)
	return title, body, nil
}
