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

func (s *EveNotificationService) renderStructure(ctx context.Context, type_, text string, timestamp time.Time) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case OwnershipTransferred:
		title.Set("Ownership transferred")
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
		out, err := s.makeStructureBaseText(ctx, d.structureTypeID, d.solarSystemID, d.structureID, d.structureName)
		if err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{d.oldCorpID, d.newCorpID, d.characterID})
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf(
			"has been transferred from %s to %s by %s.",
			makeEveEntityProfileLink(entities[d.oldCorpID]),
			makeEveEntityProfileLink(entities[d.newCorpID]),
			makeEveEntityProfileLink(entities[d.characterID]),
		)
		body.Set(out)

	case StructureAnchoring:
		title.Set("Structure anchoring")
		var data notification.StructureAnchoring
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		out += "has started anchoring."
		body.Set(out)

	case StructureDestroyed:
		title.Set("Structure destroyed")
		var data notification.StructureDestroyed
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		out += "has been destroyed."
		body.Set(out)

	case StructureFuelAlert:
		title.Set("Structure fuel alert")
		var data notification.StructureFuelAlert
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		out += "is running out of fuel in 24hrs."
		body.Set(out)

	case StructureLostShields:
		title.Set("Structure lost shields")
		var data notification.StructureLostShields
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf(
			"has lost it's shields. Armor timer ends at **%s**.",
			fromLDAPTime(data.Timestamp).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case StructureLostArmor:
		title.Set("Structure lost armor")
		var data notification.StructureLostArmor
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf(
			"has lost it's armor. Hull timer ends at **%s**.",
			fromLDAPTime(data.Timestamp).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case StructureOnline:
		title.Set("Structure online")
		var data notification.StructureOnline
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		out += "is now online."
		body.Set(out)

	case StructuresReinforcementChanged:
		var data notification.StructuresReinforcementChanged
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		typeIDs := make([]int32, 0)
		structures := make([]structureInfo, 0)
		for _, x := range data.AllStructureInfo {
			typeID := int32(x[2].(int))
			s := structureInfo{
				structureID: int64(x[0].(int)),
				name:        x[1].(string),
				typeID:      typeID,
			}
			structures = append(structures, s)
			typeIDs = append(typeIDs, typeID)
		}
		slices.SortFunc(structures, func(a structureInfo, b structureInfo) int {
			return cmp.Compare(a.name, b.name)
		})
		entities, err := s.EveUniverseService.ToEveEntities(ctx, typeIDs)
		if err != nil {
			return title, body, err
		}
		lines := make([]string, 0)
		for _, s := range structures {
			lines = append(lines, fmt.Sprintf("- %s (%s)", s.name, entities[s.typeID].Name))
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
		title.Set("Structure services are offline")
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf("has all services off-lined.\n\n%s", strings.Join(lines, "\n\n"))
		body.Set(out)

	case StructureUnanchoring:
		title.Set("Structure unanchoring")
		var data notification.StructureUnanchoring
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		due := timestamp.Add(fromLDAPDuration(data.TimeLeft))
		out += fmt.Sprintf(
			"has started un-anchoring. It will be fully un-anchored at: %s",
			due.Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case StructureUnderAttack:
		title.Set("Structure under attack")
		var data notification.StructureUnderAttack
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		attackChar, err := s.EveUniverseService.GetOrCreateEveEntityESI(ctx, data.CharID)
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf("is under attack.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s",
			makeEveEntityProfileLink(attackChar),
			makeCorporationLink(data.CorpName),
		)
		if data.AllianceName != "" {
			out += fmt.Sprintf(
				"\n\nAttacking Alliance: %s",
				makeAllianceLink(data.AllianceName),
			)
		}
		body.Set(out)

	case StructureWentHighPower:
		title.Set("Structure went high power")
		var data notification.StructureWentHighPower
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		out += "went to high power mode."
		body.Set(out)

	case StructureWentLowPower:
		title.Set("Structure went low power")
		var data notification.StructureWentLowPower
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := s.makeStructureBaseText(ctx, data.StructureTypeID, data.SolarsystemID, data.StructureID, "")
		if err != nil {
			return title, body, err
		}
		out += "went to low power mode."
		body.Set(out)

	}
	return title, body, nil
}

type structureInfo struct {
	structureID int64
	name        string
	typeID      int32
}

func (s *EveNotificationService) makeStructureBaseText(ctx context.Context, typeID, solarSystemID int32, structureID int64, structureName string) (string, error) {
	structureType, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, typeID)
	if err != nil {
		return "", err
	}
	solarSystem, err := s.EveUniverseService.GetOrCreateEveSolarSystemESI(ctx, solarSystemID)
	if err != nil {
		return "", err
	}
	var ownerLink string
	isUpwellStructure := structureType.Group.Category.ID == app.EveCategoryStructure
	if isUpwellStructure {
		structure, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, structureID)
		if err != nil {
			return "", err
		}
		if structure.Variant() == app.EveLocationStructure {
			structureName = structure.DisplayName2()
			if structure.Owner != nil {
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
	out := fmt.Sprintf("The %s in %s ", name, makeLocationLink(solarSystem))
	if ownerLink != "" {
		out += fmt.Sprintf("belonging to %s ", ownerLink)
	}
	return out, nil
}
