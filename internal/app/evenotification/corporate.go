package evenotification

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

func (s *EveNotificationService) renderCorporate(ctx context.Context, type_, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case CharAppAcceptMsg:
		var data notification.CharAppAcceptMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.CharID, data.CorpID})
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
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.CharID, data.CorpID})
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
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.CharID, data.CorpID, data.InvokingCharID})
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
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.CharID, data.CorpID})
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
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.CharID, data.CorpID})
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
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.CharID, data.CorpID})
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
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.CharID, data.CorpID})
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
