package evenotification

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
	"github.com/ErikKalkoken/evebuddy/pkg/optional"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

type structureReinforcementInfo struct {
	structureID int64
	name        string
	typeID      int32
}

func (s *EveNotificationService) renderStructure(ctx context.Context, type_ Type, text string, timestamp time.Time) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case OwnershipTransferred:
		var d struct {
			characterID     int32
			newCorpID       int32
			oldCorpID       int32
			solarSystemID   int32
			structureID     int64
			structureName   string
			structureTypeID int32
		}
		if strings.Contains(text, "newOwnerCorpID") {
			var data notification2.OwnershipTransferredV2
			if err := yaml.Unmarshal([]byte(text), &data); err != nil {
				return title, body, err
			}
			d.characterID = data.CharID
			d.newCorpID = data.NewOwnerCorpID
			d.oldCorpID = data.OldOwnerCorpID
			d.solarSystemID = data.SolarSystemID
			d.structureID = data.StructureID
			d.structureTypeID = data.StructureTypeID
			d.structureName = data.StructureName
		} else {
			var data notification.OwnershipTransferred
			if err := yaml.Unmarshal([]byte(text), &data); err != nil {
				return title, body, err
			}
			d.characterID = int32(data.CharacterLinkData[2].(int))
			d.newCorpID = int32(data.ToCorporationLinkData[2].(int))
			d.oldCorpID = int32(data.FromCorporationLinkData[2].(int))
			d.solarSystemID = int32(data.SolarSystemLinkData[2].(int))
			d.structureID = int64(data.StructureLinkData[2].(int))
			d.structureTypeID = int32(data.StructureLinkData[1].(int))
			d.structureName = data.StructureName
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{d.oldCorpID, d.newCorpID, d.characterID})
		if err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, d.structureTypeID, d.solarSystemID, d.structureID, d.structureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s ownership has been transferred to %s",
			d.structureName,
			entities[d.newCorpID].Name,
		))
		body.Set(fmt.Sprintf(
			"%s has been transferred from %s to %s by %s.",
			o.intro,
			makeEveEntityProfileLink(entities[d.oldCorpID]),
			makeEveEntityProfileLink(entities[d.newCorpID]),
			makeEveEntityProfileLink(entities[d.characterID]),
		))

	case StructureAnchoring:
		var data notification.StructureAnchoring
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"A %s has started anchoring in %s",
			o.type_.Name,
			o.solarSystem.Name,
		))
		body.Set(fmt.Sprintf("%s has started anchoring.", o.intro))

	case StructureDestroyed:
		var data notification.StructureDestroyed
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s in %s has been destroyed",
			o.name,
			o.solarSystem.Name,
		))
		body.Set(fmt.Sprintf(
			"%s has been destroyed. Item located inside the structure are available for transfer to asset safety.",
			o.intro,
		))

	case StructureFuelAlert:
		var data notification.StructureFuelAlert
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s in %s is low on fuel",
			o.name,
			o.solarSystem.Name,
		))
		body.Set(fmt.Sprintf("%s is running out of fuel in 24hrs.", o.intro))

	case StructureLostShields:
		var data notification.StructureLostShields
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s in %s has lost it's shields",
			o.name,
			o.solarSystem.Name,
		))
		body.Set(fmt.Sprintf(
			"%s has lost it's shields and is now in reinforcement state. "+
				"It will exit reinforcement at **%s** and will then be vulnerable for 15 minutes.",
			o.intro,
			fromLDAPTime(data.Timestamp).Format(app.TimeDefaultFormat),
		))

	case StructureLostArmor:
		var data notification.StructureLostArmor
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s in %s has lost it's armor",
			o.name,
			o.solarSystem.Name,
		))
		body.Set(fmt.Sprintf(
			"%s has lost it's armor. Hull timer ends at **%s**.",
			o.intro,
			fromLDAPTime(data.Timestamp).Format(app.TimeDefaultFormat),
		))

	case StructureOnline:
		var data notification.StructureOnline
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s in %s is now online",
			o.name,
			o.solarSystem.Name,
		))
		body.Set(fmt.Sprintf("%s is now online.", o.intro))

	case StructuresReinforcementChanged:
		var data notification.StructuresReinforcementChanged
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		typeIDs := make([]int32, 0)
		structures := make([]structureReinforcementInfo, 0)
		for _, x := range data.AllStructureInfo {
			typeID := int32(x[2].(int))
			s := structureReinforcementInfo{
				structureID: int64(x[0].(int)),
				name:        x[1].(string),
				typeID:      typeID,
			}
			structures = append(structures, s)
			typeIDs = append(typeIDs, typeID)
		}
		slices.SortFunc(structures, func(a structureReinforcementInfo, b structureReinforcementInfo) int {
			return cmp.Compare(a.name, b.name)
		})
		entities, err := s.EveUniverseService.ToEveEntities(ctx, typeIDs)
		if err != nil {
			return title, body, err
		}
		lines := make([]string, 0)
		for _, o := range structures {
			lines = append(lines, fmt.Sprintf("- %s (%s)", o.name, entities[o.typeID].Name))
		}
		title.Set("Structure reinforcement time changed")
		out := fmt.Sprintf(
			"Reinforcement hour has been changed to %d:00 "+
				"for the following structures:\n\n%s",
			data.Hour,
			strings.Join(lines, "\n\n"),
		)
		body.Set(out)

	case StructureServicesOffline:
		var data notification.StructureServicesOffline
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, data.ListOfServiceModuleIDs)
		if err != nil {
			return title, body, err
		}
		lines := make([]string, 0)
		for e := range maps.Values(entities) {
			lines = append(lines, fmt.Sprintf("- %s", e.Name))
		}
		slices.Sort(lines)
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s in %s has all services off-lined",
			o.name,
			o.solarSystem.Name,
		))
		body.Set(fmt.Sprintf(
			"%s has all services off-lined.\n\n%s",
			o.intro,
			strings.Join(lines, "\n\n"),
		))

	case StructureUnanchoring:
		var data notification.StructureUnanchoring
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s has started unanchoring in %s",
			o.name,
			o.solarSystem.Name,
		))
		due := timestamp.Add(fromLDAPDuration(data.TimeLeft))
		body.Set(fmt.Sprintf(
			"%s has started un-anchoring. It will be fully un-anchored at: %s",
			o.intro,
			due.Format(app.TimeDefaultFormat),
		))

	case StructureUnderAttack:
		var data notification.StructureUnderAttack
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s in %s is under attack",
			o.name,
			o.solarSystem.Name,
		))
		attackChar, err := s.EveUniverseService.GetOrCreateEveEntityESI(ctx, data.CharID)
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
		body.Set(t)

	case StructureWentHighPower:
		var data notification.StructureWentHighPower
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s is now running on High Power", o.name))
		body.Set(fmt.Sprintf("%s went to high power mode.", o.intro))

	case StructureWentLowPower:
		var data notification.StructureWentLowPower
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s is now running on Low Power", o.name))
		body.Set(fmt.Sprintf("%s went to low power mode.", o.intro))

	}
	return title, body, nil
}

type structureInfo struct {
	type_       *app.EveType
	solarSystem *app.EveSolarSystem
	owner       *app.EveEntity
	name        string
	intro       string
}

func (s *EveNotificationService) makeStructureBaseText(ctx context.Context, typeID, solarSystemID int32, structureID int64, structureName string) (structureInfo, error) {
	structureType, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, typeID)
	if err != nil {
		return structureInfo{}, err
	}
	solarSystem, err := s.EveUniverseService.GetOrCreateEveSolarSystemESI(ctx, solarSystemID)
	if err != nil {
		return structureInfo{}, err
	}
	var ownerLink string
	var owner *app.EveEntity
	isUpwellStructure := structureType.Group.Category.ID == app.EveCategoryStructure
	if isUpwellStructure {
		structure, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, structureID)
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
	isOrbital := structureType.Group.Category.ID == app.EveCategoryOrbitals
	if isOrbital && structureName != "" {
		name = fmt.Sprintf("**%s**", structureName)
	} else if structureName != "" {
		name = fmt.Sprintf("%s **%s**", structureType.Name, structureName)
	} else {
		name = structureType.Name
	}
	text := fmt.Sprintf("The %s in %s", name, makeLocationLink(solarSystem))
	if ownerLink != "" {
		text += fmt.Sprintf(" belonging to %s", ownerLink)
	}
	x := structureInfo{
		type_:       structureType,
		solarSystem: solarSystem,
		name:        structureName,
		owner:       owner,
		intro:       text,
	}
	return x, nil
}
