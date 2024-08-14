package eveuniverse

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse/notificationtype"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"github.com/dustin/go-humanize"
	"gopkg.in/yaml.v3"
)

// RenderEveNotificationESI renders title and body for a notification and return them.
func (eus *EveUniverseService) RenderEveNotificationESI(ctx context.Context, type_, text string, timestamp time.Time) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case "CorpAllBillMsg":
		title.Set("Bill issued")
		var data notificationtype.CorpAllBillMsgV2
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := eus.ToEveEntities(ctx, []int32{data.CreditorID, data.DebtorID})
		if err != nil {
			return title, body, err
		}
		var out strings.Builder
		t := template.Must(template.New(type_).Parse(
			"A bill of **{{.amount}}** ISK, due **{{.dueDate}}** owed by {{.debtor}} to {{.creditor}} " +
				"was issued on {{.currentDate}}. This bill is for {{.billType}}.",
		))

		if err := t.Execute(&out, map[string]string{
			"amount":      humanize.Commaf(data.Amount),
			"dueDate":     FromLDAPTime(data.DueDate).Format(app.TimeDefaultFormat),
			"debtor":      makeEveEntityProfileLink(entities[data.DebtorID]),
			"creditor":    makeEveEntityProfileLink(entities[data.CreditorID]),
			"currentDate": FromLDAPTime(data.CurrentDate).Format(app.TimeDefaultFormat),
			"billType":    billTypeName(data.BillTypeID),
		}); err != nil {
			return title, body, err
		}
		body.Set(out.String())

	// Structure notifications
	case "OwnershipTransferred":
		title.Set("Ownership transferred")
		var d struct {
			characterID     int32
			newCorpID       int32
			oldCorpID       int32
			solarSystemID   int32
			structureID     int64
			structureTypeID int32
		}
		if strings.Contains(text, "newOwnerCorpID") {
			var data notificationtype.OwnershipTransferredV2
			if err := yaml.Unmarshal([]byte(text), &data); err != nil {
				return title, body, err
			}
			d.characterID = data.CharID
			d.newCorpID = data.NewOwnerCorpID
			d.oldCorpID = data.OldOwnerCorpID
			d.solarSystemID = data.SolarSystemID
			d.structureID = data.StructureID
			d.structureTypeID = data.StructureTypeID
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
		}
		out, err := makeStructureBaseText(ctx, eus, d.structureTypeID, d.solarSystemID, d.structureID)
		if err != nil {
			return title, body, err
		}
		entities, err := eus.ToEveEntities(ctx, []int32{d.oldCorpID, d.newCorpID, d.characterID})
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

	case "StructureAnchoring":
		title.Set("Structure anchoring")
		var data notification.StructureAnchoring
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		out += "has started anchoring."
		body.Set(out)

	case "StructureUnderAttack":
		title.Set("Structure under attack")
		var data notification.StructureUnderAttack
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		attackChar, err := eus.GetOrCreateEveEntityESI(ctx, data.CharID)
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf("is under attack.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s\n\n"+
			"Attacking Alliance: %s",
			makeEveEntityProfileLink(attackChar),
			makeCorporationLink(data.CorpName),
			makeAllianceLink(data.AllianceName),
		)
		body.Set(out)

	case "StructureLostShields":
		title.Set("Structure lost shields")
		var data notification.StructureLostShields
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf(
			"has lost it's shields. Armor timer ends at **%s**.",
			FromLDAPTime(data.Timestamp).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case "StructureLostArmor":
		title.Set("Structure lost armor")
		var data notification.StructureLostArmor
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		out += fmt.Sprintf(
			"has lost it's armor. Hull timer ends at **%s**.",
			FromLDAPTime(data.Timestamp).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case "StructureDestroyed":
		title.Set("Structure destroyed")
		var data notification.StructureDestroyed
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		out += "has been destroyed."
		body.Set(out)

	case "StructureWentLowPower":
		title.Set("Structure went low power")
		var data notification.StructureWentLowPower
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		out += "went to low power mode."
		body.Set(out)

	case "StructureWentHighPower":
		title.Set("Structure went high power")
		var data notification.StructureWentHighPower
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		out += "went to high power mode."
		body.Set(out)

	case "StructureOnline":
		title.Set("Structure online")
		var data notification.StructureOnline
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		out += "is now online."
		body.Set(out)

	case "StructureFuelAlert":
		title.Set("Structure fuel alert")
		var data notification.StructureFuelAlert
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		out += "is running out of fuel in 24hrs."
		body.Set(out)

	case "StructureUnanchoring":
		title.Set("Structure unanchoring")
		var data notification.StructureUnanchoring
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out, err := makeStructureBaseText(ctx, eus, data.StructureTypeID, data.SolarsystemID, data.StructureID)
		if err != nil {
			return title, body, err
		}
		due := timestamp.Add(FromLDAPDuration(data.TimeLeft))
		out += fmt.Sprintf(
			"has started un-anchoring. It will be fully un-anchored at: %s",
			due.Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	}
	return title, body, nil
}

func makeStructureBaseText(ctx context.Context, s *EveUniverseService, typeID, solarSystemID int32, structureID int64) (string, error) {
	structureType, err := s.GetOrCreateEveTypeESI(ctx, typeID)
	if err != nil {
		return "", err
	}
	solarSystem, err := s.GetOrCreateEveSolarSystemESI(ctx, solarSystemID)
	if err != nil {
		return "", err
	}
	structure, err := s.GetOrCreateEveLocationESI(ctx, structureID)
	if err != nil {
		return "", err
	}
	structureName := structure.DisplayName2()
	if structureName == "" {
		structureName = "?"
	}
	out := fmt.Sprintf("The %s **%s** in %s ", structureType.Name, structureName, makeLocationLink(solarSystem))
	return out, nil
}

func billTypeName(id int32) string {
	switch id {
	case 7:
		return "Infrastructure Hub"
	}
	return "?"
}

func makeLocationLink(ess *app.EveSolarSystem) string {
	x := fmt.Sprintf(
		"%s (%s)",
		makeMarkDownLink(ess.Name, makeDotLanProfileURL(ess.Name, dotlanSolarSystem)),
		ess.Constellation.Region.Name,
	)
	return x
}

// func makeCharacterLink(id int32, name string) string {
// 	return makeMarkDownLink(name, makeEveWhoCharacterURL(id))
// }

func makeCorporationLink(name string) string {
	if name == "" {
		return ""
	}
	return makeMarkDownLink(name, makeDotLanProfileURL(name, dotlanCorporation))
}

func makeAllianceLink(name string) string {
	if name == "" {
		return ""
	}
	return makeMarkDownLink(name, makeDotLanProfileURL(name, dotlanAlliance))
}

func makeEveWhoCharacterURL(id int32) string {
	return fmt.Sprintf("https://evewho.com/character/%d", id)
}

func makeEveEntityProfileLink(e *app.EveEntity) string {
	var url string
	switch e.Category {
	case app.EveEntityAlliance:
		url = makeDotLanProfileURL(e.Name, dotlanAlliance)
	case app.EveEntityCharacter:
		url = makeEveWhoCharacterURL(e.ID)
	case app.EveEntityCorporation:
		url = makeDotLanProfileURL(e.Name, dotlanCorporation)
	}
	return makeMarkDownLink(e.Name, url)
}

func makeMarkDownLink(label, url string) string {
	return fmt.Sprintf("[%s](%s)", label, url)
}

// FromLDAPTime converts an ldap time to golang time
func FromLDAPTime(ldap_dt int64) time.Time {
	return time.Unix((ldap_dt/10000000)-11644473600, 0).UTC()
}

// FromLDAPDuration converts an ldap duration to golang duration
func FromLDAPDuration(ldap_td int64) time.Duration {
	return time.Duration(ldap_td/10) * time.Microsecond
}
