package evenotification

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

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
