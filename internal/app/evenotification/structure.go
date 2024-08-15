package evenotification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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
	}
	return title, body, nil
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
