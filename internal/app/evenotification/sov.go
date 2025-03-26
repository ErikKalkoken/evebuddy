package evenotification

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

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
	panic("Notification type not implemented: " + type_)
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
