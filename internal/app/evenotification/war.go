package evenotification

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
	"github.com/ErikKalkoken/evebuddy/pkg/optional"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

func (s *EveNotificationService) renderWar(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case AllWarSurrenderMsg:
		var data notification.AllWarSurrenderMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
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

	case CorpWarSurrenderMsg:
		var data notification.CorpWarSurrenderMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
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

	case WarAdopted:
		var data notification2.WarAdopted
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(
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

	case WarDeclared:
		var data notification2.WarDeclared
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
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

	case WarHQRemovedFromSpace:
		var data notification2.WarHQRemovedFromSpace
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
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
			fromLDAPTime(data.TimeDeclared).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	// TODO: Double-check that this render is correct
	case WarInherited:
		var data notification2.WarInherited
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(
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
			"%s inherits war against %s",
			entities[data.AllianceID].Name,
			entities[data.OpponentID].Name,
		))
		out := fmt.Sprintf(
			"%s has inherited the war between %s and "+
				"%s from newly joined %s.\n\n"+
				"Within **24** hours fighting can legally occur between those involved.",
			makeEveEntityProfileLink(entities[data.AllianceID]),
			makeEveEntityProfileLink(entities[data.DeclaredByID]),
			makeEveEntityProfileLink(entities[data.AgainstID]),
			makeEveEntityProfileLink(entities[data.QuitterID]),
		)
		body.Set(out)

	case WarInvalid:
		var data notification2.WarInvalid
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
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
			fromLDAPTime(data.EndDate).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case WarRetractedByConcord:
		var data notification2.WarRetractedByConcord
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.AgainstID, data.DeclaredByID})
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
			fromLDAPTime(data.EndDate).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	}
	return title, body, nil
}
