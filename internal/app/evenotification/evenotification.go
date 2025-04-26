// Package evenotification contains the business logic for dealing with Eve Online notifications.
// It defines the notification types and related categories
// and provides a service for rendering notifications titles and bodies.
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
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"github.com/dustin/go-humanize"
	"gopkg.in/yaml.v3"
)

// EveNotificationService is a service for rendering notifications.
type EveNotificationService struct {
	eus *eveuniverseservice.EveUniverseService
}

func New(eus *eveuniverseservice.EveUniverseService) *EveNotificationService {
	s := &EveNotificationService{eus: eus}
	return s
}

// RenderESI renders title and body for all supported notification types and returns them.
// Returns empty title and body for unsupported notification types.
func (s *EveNotificationService) RenderESI(ctx context.Context, type_, text string, timestamp time.Time) (optional.Optional[string], optional.Optional[string], error) {
	switch t := Type(type_); t {
	case BillOutOfMoneyMsg,
		BillPaidCorpAllMsg,
		CorpAllBillMsg,
		InfrastructureHubBillAboutToExpire,
		IHubDestroyedByBillFailure:
		return s.renderBilling(ctx, t, text)

	case CharAppAcceptMsg,
		CharAppRejectMsg,
		CharAppWithdrawMsg,
		CharLeftCorpMsg,
		CorpAppInvitedMsg,
		CorpAppNewMsg,
		CorpAppRejectCustomMsg:
		return s.renderCorporate(ctx, t, text)

	case OrbitalAttacked,
		OrbitalReinforced:
		return s.renderOrbital(ctx, t, text)

	case MoonminingExtractionStarted,
		MoonminingExtractionFinished,
		MoonminingAutomaticFracture,
		MoonminingExtractionCancelled,
		MoonminingLaserFired:
		return s.renderMoonMining(ctx, t, text)

	case OwnershipTransferred,
		StructureAnchoring,
		StructureDestroyed,
		StructureFuelAlert,
		StructureImpendingAbandonmentAssetsAtRisk,
		StructureItemsDelivered,
		StructureItemsMovedToSafety,
		StructureLostArmor,
		StructureLostShields,
		StructureOnline,
		StructureServicesOffline,
		StructuresReinforcementChanged,
		StructureUnanchoring,
		StructureUnderAttack,
		StructureWentHighPower,
		StructureWentLowPower:
		return s.renderStructure(ctx, t, text, timestamp)

	case TowerAlertMsg,
		TowerResourceAlertMsg:
		return s.renderTower(ctx, t, text)
	case AllWarSurrenderMsg,
		CorpWarSurrenderMsg,
		DeclareWar,
		WarAdopted,
		WarDeclared,
		WarHQRemovedFromSpace,
		WarInherited,
		WarInvalid,
		WarRetractedByConcord:
		return s.renderWar(ctx, t, text)
	case EntosisCaptureStarted,
		SovAllClaimAcquiredMsg,
		SovAllClaimLostMsg,
		SovCommandNodeEventStarted,
		SovStructureDestroyed,
		SovStructureReinforced:
		return s.renderSov(ctx, t, text)
	}
	return optional.Optional[string]{}, optional.Optional[string]{}, nil
}

const (
	billTypeLease             = 2
	billTypeAlliance          = 5
	billTypeInfrastructureHub = 7
)

func billTypeName(id int32) string {
	switch id {
	case billTypeLease:
		return "lease"
	case billTypeAlliance:
		return "alliance maintenance"
	case billTypeInfrastructureHub:
		return "infrastructure hub upkeep"
	}
	return "?"
}

func (s *EveNotificationService) renderBilling(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case BillPaidCorpAllMsg:
		title.Set("Bill payed")
		var data notification.BillPaidCorpAllMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out := fmt.Sprintf(
			"A bill of **%s** ISK, due **%s** was payed.",
			humanize.Commaf(float64(data.Amount)),
			fromLDAPTime(data.DueDate).Format(app.DateTimeFormat),
		)
		body.Set(out)

	case BillOutOfMoneyMsg:
		var data notification.CorpAllBillMsgV2
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("Insufficient funds for %s bill", billTypeName(data.BillTypeID)))
		out := fmt.Sprintf(
			"The selected corporation wallet division for automatic payments "+
				"does not have enough current funds available to pay the %s bill, "+
				"due to be paid by %s. "+
				"Transfer additional funds to the selected wallet "+
				"division in order to meet your pending automatic bills.",
			billTypeName(data.BillTypeID),
			fromLDAPTime(data.DueDate).Format(app.DateTimeFormat),
		)
		body.Set(out)

	case CorpAllBillMsg:
		var data notification.CorpAllBillMsgV2
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("Bill issued for %s", billTypeName(data.BillTypeID)))
		ids := []int32{data.CreditorID, data.DebtorID}
		if data.ExternalID != -1 && data.ExternalID == int64(int32(data.ExternalID)) {
			ids = append(ids, int32(data.ExternalID))
		}
		if data.ExternalID2 != -1 && data.ExternalID2 == int64(int32(data.ExternalID2)) {
			ids = append(ids, int32(data.ExternalID2))
		}
		entities, err := s.eus.ToEntities(ctx, ids)
		if err != nil {
			return title, body, err
		}
		var external1 string
		if x, ok := entities[int32(data.ExternalID)]; ok && x.Name != "" {
			external1 = x.Name
		} else {
			external1 = "?"
		}
		var external2 string
		if x, ok := entities[int32(data.ExternalID2)]; ok && x.Name != "" {
			external2 = x.Name
		} else {
			external2 = "?"
		}
		var billPurpose string
		switch data.BillTypeID {
		case billTypeLease:
			billPurpose = fmt.Sprintf("extending the lease of **%s** at **%s**", external1, external2)
		case billTypeAlliance:
			billPurpose = fmt.Sprintf("maintenance of **%s**", external1)
		case billTypeInfrastructureHub:
			billPurpose = fmt.Sprintf("maintenance of infrastructure hub in **%s**", external1)
		default:
			billPurpose = "?"
		}
		body.Set(fmt.Sprintf(
			"A bill of **%s** ISK, due **%s** owed by %s to %s was issued on %s. This bill is for %s.",
			humanize.Commaf(data.Amount),
			fromLDAPTime(data.DueDate).Format(app.DateTimeFormat),
			makeEveEntityProfileLink(entities[data.DebtorID]),
			makeEveEntityProfileLink(entities[data.CreditorID]),
			fromLDAPTime(data.CurrentDate).Format(app.DateTimeFormat),
			billPurpose,
		))

	case InfrastructureHubBillAboutToExpire:
		title.Set("IHub Bill About to Expire")
		var data notification.InfrastructureHubBillAboutToExpire
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("Maintenance bill for Infrastructure Hub in %s expires at %s, "+
			"if not paid in time this Infrastructure Hub will self-destruct.",
			makeSolarSystemLink(solarSystem),
			fromLDAPTime(data.DueDate).Format(app.DateTimeFormat),
		)
		body.Set(out)

	case IHubDestroyedByBillFailure:
		var data notification.IHubDestroyedByBillFailure
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
		if err != nil {
			return title, body, err
		}
		structureType, err := s.eus.GetOrCreateTypeESI(ctx, int32(data.StructureTypeID))
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s has self-destructed due to unpaid maintenance bills",
			structureType.Name,
		))
		out := fmt.Sprintf("%s in %s has self-destructed, as the standard maintenance bills where not paid.",
			structureType.Name,
			makeSolarSystemLink(solarSystem),
		)
		body.Set(out)
	}
	return title, body, nil
}

func (s *EveNotificationService) renderCorporate(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case CharAppAcceptMsg:
		var data notification.CharAppAcceptMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.eus.ToEntities(ctx, []int32{data.CharID, data.CorpID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s joins %s",
			entities[data.CharID].Name,
			entities[data.CorpID].Name,
		))
		out := fmt.Sprintf(
			"%s is now a member of %s.",
			makeEveEntityProfileLink(entities[data.CorpID]),
			makeEveEntityProfileLink(entities[data.CharID]),
		)
		body.Set(out)

	case CorpAppNewMsg:
		var data notification.CorpAppNewMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.eus.ToEntities(ctx, []int32{data.CharID, data.CorpID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("New application from %s", entities[data.CharID].Name))
		out := fmt.Sprintf(
			"New application from %s to join %s:\n\n> %s",
			makeEveEntityProfileLink(entities[data.CharID]),
			makeEveEntityProfileLink(entities[data.CorpID]),
			data.ApplicationText,
		)
		body.Set(out)

	case CorpAppInvitedMsg:
		var data notification.CorpAppInvitedMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.eus.ToEntities(ctx, []int32{data.CharID, data.CorpID, data.InvokingCharID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s has been invited", entities[data.CharID].Name))
		out := fmt.Sprintf(
			"%s has been invited to join %s by %s:\n\n> %s",
			makeEveEntityProfileLink(entities[data.CharID]),
			makeEveEntityProfileLink(entities[data.CorpID]),
			makeEveEntityProfileLink(entities[data.InvokingCharID]),
			data.ApplicationText,
		)
		body.Set(out)

	case CharAppRejectMsg:
		var data notification.CharAppRejectMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.eus.ToEntities(ctx, []int32{data.CharID, data.CorpID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s rejected invitation", entities[data.CharID].Name))
		out := fmt.Sprintf(
			"Application from %s to join %s has been rejected:\n\n> %s",
			makeEveEntityProfileLink(entities[data.CharID]),
			makeEveEntityProfileLink(entities[data.CorpID]),
			data.ApplicationText,
		)
		body.Set(out)

	case CorpAppRejectCustomMsg:
		var data notification.CorpAppRejectCustomMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.eus.ToEntities(ctx, []int32{data.CharID, data.CorpID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("Application from %s rejected", entities[data.CharID].Name))
		out := fmt.Sprintf(
			"%s has rejected application from %s:\n\n>%s",
			makeEveEntityProfileLink(entities[data.CorpID]),
			makeEveEntityProfileLink(entities[data.CharID]),
			data.ApplicationText,
		)
		if data.CustomMessage != "" {
			out += fmt.Sprintf("\n\nReply:\n\n>%s", data.CustomMessage)
		}
		body.Set(out)

	case CharAppWithdrawMsg:
		var data notification.CharAppWithdrawMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.eus.ToEntities(ctx, []int32{data.CharID, data.CorpID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s withdrew application", entities[data.CharID].Name))
		out := fmt.Sprintf(
			"%s has withdrawn application to join %s:\n\n>%s",
			makeEveEntityProfileLink(entities[data.CorpID]),
			makeEveEntityProfileLink(entities[data.CharID]),
			data.ApplicationText,
		)
		body.Set(out)

	case CharLeftCorpMsg:
		var data notification.CharLeftCorpMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.eus.ToEntities(ctx, []int32{data.CharID, data.CorpID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s left %s",
			entities[data.CharID].Name,
			entities[data.CorpID].Name,
		))
		out := fmt.Sprintf(
			"%s is no longer a member of %s.",
			makeEveEntityProfileLink(entities[data.CorpID]),
			makeEveEntityProfileLink(entities[data.CharID]),
		)
		body.Set(out)
	}
	return title, body, nil
}

func (s *EveNotificationService) renderMoonMining(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case MoonminingAutomaticFracture:
		var data notification.MoonminingAutomaticFracture
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("Extraction for %s has autofractured", data.StructureName))
		ores, err := s.makeOreText(ctx, data.OreVolumeByType)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("The extraction for %s "+
			"has reached the end of it's lifetime and has fractured automatically. The moon products are ready to be harvested.\n\n%s",
			o.text,
			ores,
		)
		body.Set(out)

	case MoonminingExtractionStarted:
		var data notification.MoonminingExtractionStarted
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("Extraction started at %s", data.StructureName))
		ores, err := s.makeOreText(ctx, data.OreVolumeByType)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("A moon mining extraction has been started %s.\n\n"+
			"The chunk will be ready on location at %s, "+
			"and will fracture automatically on %s.\n\n%s",
			o.text,
			fromLDAPTime(data.ReadyTime).Format(app.DateTimeFormat),
			fromLDAPTime(data.AutoTime).Format(app.DateTimeFormat),
			ores,
		)
		body.Set(out)

	case MoonminingExtractionFinished:
		var data notification.MoonminingExtractionFinished
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("Extraction finished at %s", data.StructureName))
		ores, err := s.makeOreText(ctx, data.OreVolumeByType)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("The extraction %s "+
			"is finished and the chunk is ready to be shot at.\n\n"+
			"The chunk will automatically fracture on %s.\n\n%s",
			o.text,
			fromLDAPTime(data.AutoTime).Format(app.DateTimeFormat),
			ores,
		)
		body.Set(out)

	case MoonminingExtractionCancelled:
		var data notification.MoonminingExtractionCancelled
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("Extraction canceled at %s", data.StructureName))
		cancelledBy := ""
		if data.CancelledBy != 0 {
			x, err := s.eus.GetOrCreateEntityESI(ctx, data.CancelledBy)
			if err != nil {
				return title, body, err
			}
			cancelledBy = fmt.Sprintf(" by %s", makeEveEntityProfileLink(x))
		}
		out := fmt.Sprintf(
			"An ongoing extraction for %s has been cancelled%s.",
			o.text,
			cancelledBy,
		)
		body.Set(out)

	case MoonminingLaserFired:
		var data notification.MoonminingLaserFired
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s has fired it's moon drill", data.StructureName))
		firedBy := ""
		if data.FiredBy != 0 {
			x, err := s.eus.GetOrCreateEntityESI(ctx, data.FiredBy)
			if err != nil {
				return title, body, err
			}
			firedBy = fmt.Sprintf("by %s ", makeEveEntityProfileLink(x))
		}

		ores, err := s.makeOreText(ctx, data.OreVolumeByType)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf(
			"The moon drill fitted to %s has been fired %s"+
				"and the moon products are ready to be harvested.\n\n%s",
			o.text,
			firedBy,
			ores,
		)
		body.Set(out)
	}
	return title, body, nil
}

type moonMiningInfo struct {
	moon *app.EveMoon
	text string
}

func (s *EveNotificationService) makeMoonMiningBaseText(ctx context.Context, moonID int32, structureName string) (moonMiningInfo, error) {
	moon, err := s.eus.GetOrCreateMoonESI(ctx, moonID)
	if err != nil {
		return moonMiningInfo{}, err
	}
	text := fmt.Sprintf(
		"for **%s** at %s in %s",
		structureName,
		moon.Name,
		makeSolarSystemLink(moon.SolarSystem),
	)
	x := moonMiningInfo{
		moon: moon,
		text: text,
	}
	return x, nil
}

type oreItem struct {
	id     int32
	name   string
	volume float64
}

func (s *EveNotificationService) makeOreText(ctx context.Context, ores map[int32]float64) (string, error) {
	ids := slices.Collect(maps.Keys(ores))
	entities, err := s.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", err
	}
	items := make([]oreItem, 0)
	for id, v := range ores {
		i := oreItem{
			id:     id,
			name:   entities[id].Name,
			volume: v,
		}
		items = append(items, i)
	}
	slices.SortFunc(items, func(a, b oreItem) int {
		return cmp.Compare(a.name, b.name)
	})
	lines := []string{"Estimated ore composition:"}
	for i := range slices.Values(items) {
		text := fmt.Sprintf("%s: %s m3", i.name, humanize.Comma(int64(i.volume)))
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n\n"), nil
}

func (s *EveNotificationService) renderOrbital(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case OrbitalAttacked:
		var data notification.OrbitalAttacked
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeOrbitalBaseText(ctx, data.PlanetID, data.TypeID)
		if err != nil {
			return title, body, err
		}
		entities, err := s.eus.ToEntities(ctx, []int32{data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s at %s is under attack",
			o.type_.Name,
			o.planet.Name,
		))
		t := fmt.Sprintf("%s is under attack.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s",
			o.intro,
			makeEveEntityProfileLink(entities[data.AggressorID]),
			makeEveEntityProfileLink(entities[data.AggressorCorpID]),
		)
		if data.AggressorAllianceID != 0 {
			t += fmt.Sprintf(
				"\n\nAttacking Alliance: %s",
				makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
			)
		}
		body.Set(t)

	case OrbitalReinforced:
		var data notification.OrbitalReinforced
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeOrbitalBaseText(ctx, data.PlanetID, data.TypeID)
		if err != nil {
			return title, body, err
		}
		entities, err := s.eus.ToEntities(ctx, []int32{data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID})
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s at %s has been reinforced",
			o.type_.Name,
			o.planet.Name,
		))
		t := fmt.Sprintf("has been reinforced and will come out at %s.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s",
			fromLDAPTime(data.ReinforceExitTime).Format(app.DateTimeFormat),
			makeEveEntityProfileLink(entities[data.AggressorID]),
			makeEveEntityProfileLink(entities[data.AggressorCorpID]),
		)
		if data.AggressorAllianceID != 0 {
			t += fmt.Sprintf(
				"\n\nAttacking Alliance: %s",
				makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
			)
		}
		body.Set(t)
	}
	return title, body, nil
}

type orbitalInfo struct {
	type_  *app.EveType
	planet *app.EvePlanet
	intro  string
}

func (s *EveNotificationService) makeOrbitalBaseText(ctx context.Context, planetID, typeID int32) (orbitalInfo, error) {
	structureType, err := s.eus.GetOrCreateTypeESI(ctx, typeID)
	if err != nil {
		return orbitalInfo{}, err
	}
	planet, err := s.eus.GetOrCreatePlanetESI(ctx, planetID)
	if err != nil {
		return orbitalInfo{}, err
	}
	into := fmt.Sprintf(
		"The %s at %s in %s ",
		structureType.Name,
		planet.Name,
		makeSolarSystemLink(planet.SolarSystem),
	)
	x := orbitalInfo{
		type_:  structureType,
		planet: planet,
		intro:  into,
	}
	return x, nil
}

func (s *EveNotificationService) renderSov(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	switch type_ {
	case EntosisCaptureStarted:
		return s.renderEntosisCaptureStarted(ctx, text)
	case SovAllClaimAcquiredMsg:
		return s.renderSovAllClaimAcquiredMsg(ctx, text)
	case SovAllClaimLostMsg:
		return s.renderSovAllClaimLostMsg(ctx, text)
	case SovCommandNodeEventStarted:
		return s.renderSovCommandNodeEventStarted(ctx, text)
	case SovStructureDestroyed:
		return s.renderSovStructureDestroyed(ctx, text)
	case SovStructureReinforced:
		return s.renderSovStructureReinforced(ctx, text)
	}
	return optional.Optional[string]{}, optional.Optional[string]{}, fmt.Errorf("render sov: unknown notification type: %s", type_)
}

func (s *EveNotificationService) renderEntosisCaptureStarted(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.EntosisCaptureStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	structureType, err := s.eus.GetOrCreateEntityESI(ctx, data.StructureTypeID)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("%s in %s is being captured", structureType.Name, solarSystem.Name))
	body.Set(fmt.Sprintf(
		"A capsuleer has started to influence the **%s** in %s with an Entosis Link.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
	))
	return title, body, nil
}

func (s *EveNotificationService) renderSovAllClaimAcquiredMsg(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.SovAllClaimAquiredMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	corporation, err := s.eus.GetOrCreateEntityESI(ctx, data.CorpID)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("DED Sovereignty claim acknowledgement: %s", solarSystem.Name))
	body.Set(fmt.Sprintf(
		"This mail is your confirmation that DED now officially acknowledges "+
			"that your member organization %s has claimed sovereignty "+
			"on your behalf in the system %s.",
		makeEveEntityProfileLink(corporation),
		makeSolarSystemLink(solarSystem),
	))
	return title, body, nil
}

func (s *EveNotificationService) renderSovCommandNodeEventStarted(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.SovCommandNodeEventStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	structureTypeName, err := s.eventTypeIDToName(ctx, data.CampaignEventType)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf(
		"Command nodes for %s in %s have begun to decloak",
		structureTypeName,
		solarSystem.Name,
	))
	body.Set(fmt.Sprintf(
		"Command nodes for %s in %s can now be found throughout the **%s** constellation",
		structureTypeName,
		makeSolarSystemLink(solarSystem),
		solarSystem.Constellation.Name,
	))
	return title, body, nil
}

func (s *EveNotificationService) renderSovAllClaimLostMsg(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.SovAllClaimLostMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	corporation, err := s.eus.GetOrCreateEntityESI(ctx, data.CorpID)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("Lost sovereignty in: %s", solarSystem.Name))
	body.Set(fmt.Sprintf(
		"DED acknowledges that your member organization %s has lost its claim "+
			"to sovereignty on your behalf in the system %s.",
		makeEveEntityProfileLink(corporation),
		makeSolarSystemLink(solarSystem),
	))
	return title, body, nil
}

func (s *EveNotificationService) renderSovStructureDestroyed(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.SovStructureDestroyed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	structureType, err := s.eus.GetOrCreateEntityESI(ctx, data.StructureTypeID)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("%s in %s has been destroyed", structureType.Name, solarSystem.Name))
	body.Set(fmt.Sprintf(
		"The command nodes for %s in %s have been destroyed by hostile forces.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
	))
	return title, body, nil
}

func (s *EveNotificationService) renderSovStructureReinforced(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.SovStructureReinforced
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	structureTypeName, err := s.eventTypeIDToName(ctx, data.CampaignEventType)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("%s in %s has entered reinforced mode", structureTypeName, solarSystem.Name))
	body.Set(fmt.Sprintf(
		"The %s in %s has been reinforced by hostile forces "+
			"and command nodes will begin decloaking at **%s**.",
		structureTypeName,
		makeSolarSystemLink(solarSystem),
		fromLDAPTime(data.DecloakTime).Format(app.DateTimeFormat),
	))
	return title, body, nil
}

// Returns a structure name for an event type ID.
func (s *EveNotificationService) eventTypeIDToName(ctx context.Context, eventType int32) (string, error) {
	var typeID int32
	switch eventType {
	case 1:
		typeID = app.EveTypeTCU
	case 2:
		typeID = app.EveTypeIHUB
	default:
		return "?", nil
	}
	structureType, err := s.eus.GetOrCreateEntityESI(ctx, typeID)
	if err != nil {
		return "", err
	}
	return structureType.Name, nil
}

func (s *EveNotificationService) renderStructure(ctx context.Context, type_ Type, text string, timestamp time.Time) (optional.Optional[string], optional.Optional[string], error) {
	switch type_ {
	case OwnershipTransferred:
		return s.renderOwnershipTransferred(ctx, text)
	case StructureAnchoring:
		return s.renderStructureAnchoring(ctx, text)
	case StructureDestroyed:
		return s.renderStructureDestroyed(ctx, text)
	case StructureFuelAlert:
		return s.renderStructureFuelAlert(ctx, text)
	case StructureImpendingAbandonmentAssetsAtRisk:
		return s.renderStructureImpendingAbandonmentAssetsAtRisk(ctx, text)
	case StructureItemsDelivered:
		return s.renderStructureItemsDelivered(ctx, text)
	case StructureItemsMovedToSafety:
		return s.renderStructureItemsMovedToSafety(ctx, text)
	case StructureLostArmor:
		return s.renderStructureLostArmor(ctx, text)
	case StructureLostShields:
		return s.renderStructureLostShields(ctx, text)
	case StructureOnline:
		return s.renderStructureOnline(ctx, text)
	case StructuresReinforcementChanged:
		return s.renderStructuresReinforcementChanged(ctx, text)
	case StructureServicesOffline:
		return s.renderStructureServicesOffline(ctx, text)
	case StructureUnanchoring:
		return s.renderStructureUnanchoring(ctx, text, timestamp)
	case StructureUnderAttack:
		return s.renderStructureUnderAttack(ctx, text)
	case StructureWentHighPower:
		return s.renderStructureWentHighPower(ctx, text)
	case StructureWentLowPower:
		return s.renderStructureWentLowPower(ctx, text)
	}
	return optional.Optional[string]{}, optional.Optional[string]{}, fmt.Errorf("render structure: unknown notification type: %s", type_)
}

func (s *EveNotificationService) renderOwnershipTransferred(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
		var data notification.OwnershipTransferredV2
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
	entities, err := s.eus.ToEntities(ctx, []int32{d.oldCorpID, d.newCorpID, d.characterID})
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
	return title, body, nil
}

func (s *EveNotificationService) renderStructureAnchoring(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
		o.eveType.Name,
		o.solarSystem.Name,
	))
	body.Set(fmt.Sprintf("%s has started anchoring.", o.intro))
	return title, body, nil
}

func (s *EveNotificationService) renderStructureDestroyed(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
	return title, body, nil
}

func (s *EveNotificationService) renderStructureFuelAlert(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
	return title, body, nil
}

func (s *EveNotificationService) renderStructureImpendingAbandonmentAssetsAtRisk(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.StructureImpendingAbandonmentAssetsAtRisk
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	name := evehtml.Strip(data.StructureLink)
	title.Set(fmt.Sprintf("Your assets located in %s are at risk", name))
	body.Set(fmt.Sprintf(
		"You have assets located at **%s** in %s. "+
			"These assets are at risk of loss as the structure is close to becoming abandoned.\n\n"+
			"In approximately %d days this structure will become abandoned.",
		name,
		makeSolarSystemLink(solarSystem),
		data.DaysUntilAbandon,
	))
	return title, body, nil
}

func (s *EveNotificationService) renderStructureItemsDelivered(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.StructureItemsDelivered
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	ids := []int32{data.CharID, data.StructureTypeID}
	for _, r := range data.ListOfTypesAndQty {
		ids = append(ids, r[1])
	}
	entities, err := s.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarsystemID)
	if err != nil {
		return title, body, err
	}
	structure, err := s.eus.GetOrCreateLocationESI(ctx, data.StructureID)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("Items delivered from %s", entities[data.CharID].Name))
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
	body.Set(b)
	return title, body, nil
}

func (s *EveNotificationService) renderStructureItemsMovedToSafety(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.StructureItemsMovedToSafety
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	solarSystem, err := s.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	station, err := s.eus.GetOrCreateEntityESI(ctx, data.NewStationID)
	if err != nil {
		return title, body, err
	}
	name := evehtml.Strip(data.StructureLink)
	title.Set(fmt.Sprintf("Your assets located in %s have been moved to asset safety", name))
	body.Set(fmt.Sprintf(
		"You assets located at **%s** in %s have been moved to asset safety.\n\n"+
			"They can be moved to a location of your choosing earliest at %s.\n\n"+
			"They will be moved automatically to %s by %s.",
		name,
		makeSolarSystemLink(solarSystem),
		fromLDAPTime(data.AssetSafetyMinimumTimestamp).Format(app.DateTimeFormat),
		station.Name,
		fromLDAPTime(data.AssetSafetyFullTimestamp).Format(app.DateTimeFormat),
	))
	return title, body, nil
}

func (s *EveNotificationService) renderStructureLostArmor(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
		fromLDAPTime(data.Timestamp).Format(app.DateTimeFormat),
	))
	return title, body, nil
}

func (s *EveNotificationService) renderStructureLostShields(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
		fromLDAPTime(data.Timestamp).Format(app.DateTimeFormat),
	))
	return title, body, nil
}

func (s *EveNotificationService) renderStructureOnline(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
	return title, body, nil
}

type structureReinforcementInfo struct {
	structureID int64
	name        string
	typeID      int32
}

func (s *EveNotificationService) renderStructuresReinforcementChanged(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
	entities, err := s.eus.ToEntities(ctx, typeIDs)
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
	return title, body, nil
}

func (s *EveNotificationService) renderStructureServicesOffline(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.StructureServicesOffline
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, data.ListOfServiceModuleIDs)
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
	return title, body, nil
}

func (s *EveNotificationService) renderStructureUnanchoring(ctx context.Context, text string, timestamp time.Time) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
		due.Format(app.DateTimeFormat),
	))
	return title, body, nil
}

func (s *EveNotificationService) renderStructureUnderAttack(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
	attackChar, err := s.eus.GetOrCreateEntityESI(ctx, data.CharID)
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
	return title, body, nil
}

func (s *EveNotificationService) renderStructureWentHighPower(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
	return title, body, nil
}

func (s *EveNotificationService) renderStructureWentLowPower(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
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
	return title, body, nil
}

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

func (s *EveNotificationService) makeStructureBaseText(ctx context.Context, typeID, systemID int32, structureID int64, structureName string) (structureInfo, error) {
	var eveType *app.EveType
	var err error
	if typeID != 0 {
		eveType, err = s.eus.GetOrCreateTypeESI(ctx, typeID)
		if err != nil {
			return structureInfo{}, err
		}
	}
	system, err := s.eus.GetOrCreateSolarSystemESI(ctx, systemID)
	if err != nil {
		return structureInfo{}, err
	}
	var ownerLink string
	var owner *app.EveEntity
	isUpwellStructure := eveType != nil && eveType.Group.Category.ID == app.EveCategoryStructure
	if isUpwellStructure {
		structure, err := s.eus.GetOrCreateLocationESI(ctx, structureID)
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

func (s *EveNotificationService) renderTower(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	switch type_ {
	case TowerAlertMsg:
		return s.renderTowerAlertMsg(ctx, text)
	case TowerResourceAlertMsg:
		return s.renderTowerResourceAlertMsg(ctx, text)
	}
	return optional.Optional[string]{}, optional.Optional[string]{}, fmt.Errorf("render tower: unknown notification type: %s", type_)
}

func (s *EveNotificationService) renderTowerAlertMsg(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.TowerAlertMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := s.makeTowerBaseText(ctx, data.MoonID, data.TypeID)
	if err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, []int32{data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID})
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("Starbase at %s is under attack", o.moon.Name))
	b := fmt.Sprintf(
		"%s is under attack.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s",
		o.intro,
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.AggressorCorpID]),
	)
	if data.AggressorAllianceID != 0 {
		b += fmt.Sprintf(
			"\n\nAttacking Alliance: %s",
			makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
		)
	}
	body.Set(b)
	return title, body, nil
}

func (s *EveNotificationService) renderTowerResourceAlertMsg(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.TowerResourceAlertMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := s.makeTowerBaseText(ctx, data.MoonID, data.TypeID)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("Starbase at %s is running out of fuel", o.moon.Name))
	b := fmt.Sprintf("%s is running out of fuel in less then 24hrs.\n\n", o.intro)
	if len(data.Wants) > 0 {
		b += fmt.Sprintf(
			"Fuel remaining: %s units", humanize.Comma(int64(data.Wants[0].Quantity)),
		)
	}
	body.Set(b)
	return title, body, nil
}

type towerInfo struct {
	type_ *app.EveType
	moon  *app.EveMoon
	intro string
}

func (s *EveNotificationService) makeTowerBaseText(ctx context.Context, moonID, typeID int32) (towerInfo, error) {
	structureType, err := s.eus.GetOrCreateTypeESI(ctx, typeID)
	if err != nil {
		return towerInfo{}, err
	}
	moon, err := s.eus.GetOrCreateMoonESI(ctx, moonID)
	if err != nil {
		return towerInfo{}, err
	}
	intro := fmt.Sprintf("The %s at %s in %s ", structureType.Name, moon.Name, makeSolarSystemLink(moon.SolarSystem))
	x := towerInfo{
		type_: structureType,
		moon:  moon,
		intro: intro,
	}
	return x, nil
}

func (s *EveNotificationService) renderWar(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	switch type_ {
	case AllWarSurrenderMsg:
		return s.renderAllWarSurrenderMsg(ctx, text)
	case CorpWarSurrenderMsg:
		return s.renderCorpWarSurrenderMsg(ctx, text)
	case DeclareWar:
		return s.renderDeclareWar(ctx, text)
	case WarAdopted:
		return s.renderWarAdopted(ctx, text)
	case WarDeclared:
		return s.renderWarDeclared(ctx, text)
	case WarHQRemovedFromSpace:
		return s.renderWarHQRemovedFromSpace(ctx, text)
	case WarInherited:
		return s.renderWarInherited(ctx, text)
	case WarInvalid:
		return s.renderWarInvalid(ctx, text)
	case WarRetractedByConcord:
		return s.renderWarRetractedByConcord(ctx, text)
	}
	return optional.Optional[string]{}, optional.Optional[string]{}, fmt.Errorf("render war: unknown notification type: %s", type_)
}

func (s *EveNotificationService) renderAllWarSurrenderMsg(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.AllWarSurrenderMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf(
		"%s has surrendered in the war against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	))
	out := fmt.Sprintf(
		"%s has surrendered in the war against %s.\n\n"+
			"The war will be declared as being over after approximately %d hours.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		data.DelayHours,
	)
	body.Set(out)
	return title, body, nil
}

func (s *EveNotificationService) renderCorpWarSurrenderMsg(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.CorpWarSurrenderMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
	if err != nil {
		return title, body, err
	}
	title.Set("One party has surrendered")
	out := fmt.Sprintf(
		"The war between %s and %s is coming to an end as one party has surrendered.\n\n"+
			"The war will be declared as being over after approximately 24 hours.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	body.Set(out)
	return title, body, nil
}

func (s *EveNotificationService) renderDeclareWar(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.DeclareWar
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, []int32{data.CharID, data.DefenderID, data.EntityID})
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("%s declared war", entities[data.EntityID].Name))
	out := fmt.Sprintf(
		"%s has declared war on %s on behalf of %s.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
		makeEveEntityProfileLink(entities[data.EntityID]),
	)
	body.Set(out)
	return title, body, nil
}

func (s *EveNotificationService) renderWarAdopted(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.WarAdopted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(
		ctx, []int32{data.AgainstID, data.DeclaredByID, data.AllianceID},
	)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf(
		"War update: %s has left %s",
		entities[data.AgainstID].Name,
		entities[data.AllianceID].Name,
	))
	declaredBy := makeEveEntityProfileLink(entities[data.DeclaredByID])
	alliance := makeEveEntityProfileLink(entities[data.AllianceID])
	against := makeEveEntityProfileLink(entities[data.AgainstID])
	out := fmt.Sprintf(
		"There has been a development in the war between %s and %s.\n"+
			"%s is no longer a member of %s, "+
			"and therefore a new war between %s and %s has begun.",
		declaredBy,
		alliance,
		against,
		alliance,
		declaredBy,
		alliance,
	)
	body.Set(out)
	return title, body, nil
}

func (s *EveNotificationService) renderWarDeclared(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.WarDeclared
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf(
		"%s Declares War Against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	))
	out := fmt.Sprintf(
		"%s has declared war on %s with **%s** "+
			"as the designated war headquarters.\n\n"+
			"Within **%d** hours fighting can legally occur between those involved.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		data.WarHQ,
		data.DelayHours,
	)
	body.Set(out)
	return title, body, nil
}

func (s *EveNotificationService) renderWarHQRemovedFromSpace(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.WarHQRemovedFromSpace
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("WarHQ %s lost", data.WarHQ))
	out := fmt.Sprintf(
		"The war HQ **%s** is no more. "+
			"As a consequence, the war declared by %s against %s on %s "+
			"has been declared invalid by CONCORD and has entered its cooldown period.",
		data.WarHQ,
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		fromLDAPTime(data.TimeDeclared).Format(app.DateTimeFormat),
	)
	body.Set(out)
	return title, body, nil
}

func (s *EveNotificationService) renderWarInherited(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.WarInherited
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(
		ctx,
		[]int32{
			data.AgainstID,
			data.AllianceID,
			data.DeclaredByID,
			data.OpponentID,
			data.QuitterID,
		},
	)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf(
		"War update: %s has left %s",
		entities[data.QuitterID].Name,
		entities[data.AllianceID].Name,
	))
	alliance := makeEveEntityProfileLink(entities[data.AllianceID])
	against := makeEveEntityProfileLink(entities[data.AgainstID])
	quitter := makeEveEntityProfileLink(entities[data.QuitterID])
	out := fmt.Sprintf(
		"There has been a development in the war between %s and %s.\n\n"+
			"%s is no longer a member of %s, and therefore a new war between %s and %s has begun.",
		alliance,
		against,
		quitter,
		alliance,
		against,
		quitter,
	)
	body.Set(out)
	return title, body, nil
}

func (s *EveNotificationService) renderWarInvalid(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.WarInvalid
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
	if err != nil {
		return title, body, err
	}
	title.Set("CONCORD invalidates war")
	out := fmt.Sprintf(
		"The war between %s and %s "+
			"has been invalidated by CONCORD, "+
			"because at least one of the involved parties "+
			"has become ineligible for war declarations.\n\n"+
			"Fighting must cease on %s.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		fromLDAPTime(data.EndDate).Format(app.DateTimeFormat),
	)
	body.Set(out)
	return title, body, nil
}

func (s *EveNotificationService) renderWarRetractedByConcord(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.WarRetractedByConcord
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
	if err != nil {
		return title, body, err
	}
	title.Set("CONCORD retracts war")
	out := fmt.Sprintf(
		"The war between %s and %s "+
			"has been retracted by CONCORD. \n\n"+
			"After %s CONCORD will again respond to any hostilities "+
			"between those involved with full force.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		fromLDAPTime(data.EndDate).Format(app.DateTimeFormat),
	)
	body.Set(out)
	return title, body, nil
}

// fromLDAPTime converts an ldap time to golang time
func fromLDAPTime(ldap_dt int64) time.Time {
	return time.Unix((ldap_dt/10000000)-11644473600, 0).UTC()
}

// fromLDAPDuration converts an ldap duration to golang duration
func fromLDAPDuration(ldap_td int64) time.Duration {
	return time.Duration(ldap_td/10) * time.Microsecond
}

type dotlanType = uint

const (
	dotlanAlliance dotlanType = iota
	dotlanCorporation
	dotlanSolarSystem
	dotlanRegion
)

func makeDotLanProfileURL(name string, typ dotlanType) string {
	const baseURL = "https://evemaps.dotlan.net"
	var path string
	m := map[dotlanType]string{
		dotlanAlliance:    "alliance",
		dotlanCorporation: "corp",
		dotlanSolarSystem: "system",
		dotlanRegion:      "region",
	}
	path, ok := m[typ]
	if !ok {
		return name
	}
	name2 := strings.ReplaceAll(name, " ", "_")
	return fmt.Sprintf("%s/%s/%s", baseURL, path, name2)
}

func makeSolarSystemLink(ess *app.EveSolarSystem) string {
	x := fmt.Sprintf(
		"%s (%s)",
		makeMarkDownLink(ess.Name, makeDotLanProfileURL(ess.Name, dotlanSolarSystem)),
		ess.Constellation.Region.Name,
	)
	return x
}

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
	if e == nil {
		return ""
	}
	var url string
	switch e.Category {
	case app.EveEntityAlliance:
		url = makeDotLanProfileURL(e.Name, dotlanAlliance)
	case app.EveEntityCharacter:
		url = makeEveWhoCharacterURL(e.ID)
	case app.EveEntityCorporation:
		url = makeDotLanProfileURL(e.Name, dotlanCorporation)
	default:
		return e.Name
	}
	return makeMarkDownLink(e.Name, url)
}

func makeMarkDownLink(label, url string) string {
	return fmt.Sprintf("[%s](%s)", label, url)
}
