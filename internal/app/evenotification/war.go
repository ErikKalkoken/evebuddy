package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
)

type allWarSurrenderMsg struct {
	baseRenderer
}

func (n allWarSurrenderMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allWarSurrenderMsg) unmarshal(text string) (goesi.AllWarSurrenderMsg, set.Set[int64], error) {
	var data goesi.AllWarSurrenderMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n allWarSurrenderMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf(
		"%s has surrendered in the war against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body := fmt.Sprintf(
		"%s has surrendered in the war against %s.\n\n"+
			"The war will be declared as being over after approximately %d hours.",
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
		data.DelayHours,
	)
	return title, body, nil
}

type mutualWarInviteSent struct {
	baseRenderer
}

func (n mutualWarInviteSent) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n mutualWarInviteSent) unmarshal(text string) (notification2.MutualWarInviteSent, set.Set[int64], error) {
	var data notification2.MutualWarInviteSent
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n mutualWarInviteSent) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf(
		"Mutual war invite sent to %s",
		entities[data.AgainstID].Name,
	)
	body := fmt.Sprintf(
		"%s has sent a mutual war invitation to %s.\n\n"+
			"The invitation expires on %s.",
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
		fromLDAPTime(data.ExpireTimeStamp).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type corpWarSurrenderMsg struct {
	baseRenderer
}

func (n corpWarSurrenderMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpWarSurrenderMsg) unmarshal(text string) (goesi.CorpWarSurrenderMsg, set.Set[int64], error) {
	var data goesi.CorpWarSurrenderMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n corpWarSurrenderMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "One party has surrendered"
	out := fmt.Sprintf(
		"The war between %s and %s is coming to an end as one party has surrendered.\n\n"+
			"The war will be declared as being over after approximately 24 hours.",
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
	)
	body = out
	return title, body, nil
}

type declareWar struct {
	baseRenderer
}

func (n declareWar) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n declareWar) unmarshal(text string) (goesi.DeclareWar, set.Set[int64], error) {
	var data goesi.DeclareWar
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.DefenderID, data.EntityID)
	return data, ids, nil
}

func (n declareWar) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s declared war", entities[data.EntityID].Name)
	out := fmt.Sprintf(
		"%s has declared war on %s on behalf of %s.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.DefenderID]),
		makeInfoLink(entities[data.EntityID]),
	)
	body = out
	return title, body, nil
}

type warAdopted struct {
	baseRenderer
}

func (n warAdopted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warAdopted) unmarshal(text string) (goesi.WarAdopted, set.Set[int64], error) {
	var data goesi.WarAdopted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID, data.AllianceID)
	return data, ids, nil
}

func (n warAdopted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"War update: %s has left %s",
		entities[data.AgainstID].Name,
		entities[data.AllianceID].Name,
	)
	declaredBy := makeInfoLink(entities[data.DeclaredByID])
	alliance := makeInfoLink(entities[data.AllianceID])
	against := makeInfoLink(entities[data.AgainstID])
	out := fmt.Sprintf(
		"There has been a development in the war between %s and %s.\n"+
			"%s is no longer a member of %s, "+
			"and therefore a new war between %s and %s has begun.",
		declaredBy,
		alliance,
		against,
		alliance,
		declaredBy,
		against,
	)
	body = out
	return title, body, nil
}

type warDeclared struct {
	baseRenderer
}

func (n warDeclared) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warDeclared) unmarshal(text string) (goesi.WarDeclared, set.Set[int64], error) {
	var data goesi.WarDeclared
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warDeclared) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s Declares War Against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	out := fmt.Sprintf(
		"%s has declared war on %s with **%s** "+
			"as the designated war headquarters.\n\n"+
			"Within **%d** hours fighting can legally occur between those involved.",
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
		data.WarHQ,
		data.DelayHours,
	)
	body = out
	return title, body, nil
}

type warHQRemovedFromSpace struct {
	baseRenderer
}

func (n warHQRemovedFromSpace) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warHQRemovedFromSpace) unmarshal(text string) (goesi.WarHQRemovedFromSpace, set.Set[int64], error) {
	var data goesi.WarHQRemovedFromSpace
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warHQRemovedFromSpace) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("WarHQ %s lost", data.WarHQ)
	out := fmt.Sprintf(
		"The war HQ **%s** is no more. "+
			"As a consequence, the war declared by %s against %s on %s "+
			"has been declared invalid by CONCORD and has entered its cooldown period.",
		data.WarHQ,
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
		fromLDAPTime(data.TimeDeclared).Format(app.DateTimeFormat),
	)
	body = out
	return title, body, nil
}

type warInherited struct {
	baseRenderer
}

func (n warInherited) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warInherited) unmarshal(text string) (goesi.WarInherited, set.Set[int64], error) {
	var data goesi.WarInherited
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(
		data.AgainstID,
		data.AllianceID,
		data.DeclaredByID,
		data.OpponentID,
		data.QuitterID,
	)
	return data, ids, nil
}

func (n warInherited) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"War update: %s has left %s",
		entities[data.QuitterID].Name,
		entities[data.AllianceID].Name,
	)
	alliance := makeInfoLink(entities[data.AllianceID])
	against := makeInfoLink(entities[data.AgainstID])
	quitter := makeInfoLink(entities[data.QuitterID])
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
	body = out
	return title, body, nil
}

type warInvalid struct {
	baseRenderer
}

func (n warInvalid) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warInvalid) unmarshal(text string) (goesi.WarInvalid, set.Set[int64], error) {
	var data goesi.WarInvalid
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warInvalid) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "CONCORD invalidates war"
	out := fmt.Sprintf(
		"The war between %s and %s "+
			"has been invalidated by CONCORD, "+
			"because at least one of the involved parties "+
			"has become ineligible for war declarations.\n\n"+
			"Fighting must cease on %s.",
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
		fromLDAPTime(data.EndDate).Format(app.DateTimeFormat),
	)
	body = out
	return title, body, nil
}

type warRetractedByConcord struct {
	baseRenderer
}

func (n warRetractedByConcord) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warRetractedByConcord) unmarshal(text string) (goesi.WarRetractedByConcord, set.Set[int64], error) {
	var data goesi.WarRetractedByConcord
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warRetractedByConcord) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "CONCORD retracts war"
	out := fmt.Sprintf(
		"The war between %s and %s "+
			"has been retracted by CONCORD.\n\n"+
			"After %s CONCORD will again respond to any hostilities "+
			"between those involved with full force.",
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
		fromLDAPTime(data.EndDate).Format(app.DateTimeFormat),
	)
	body = out
	return title, body, nil
}

type acceptedAlly struct {
	baseRenderer
}

func (n acceptedAlly) unmarshal(text string) (goesi.AcceptedAlly, set.Set[int64], error) {
	var data goesi.AcceptedAlly
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AllyID, data.CharID, data.EnemyID), nil
}

func (n acceptedAlly) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n acceptedAlly) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has accepted an ally offer", entities[data.AllyID].Name)
	body := fmt.Sprintf(
		"%s has accepted an offer from %s to join as an ally in the war against %s.",
		makeInfoLink(entities[data.AllyID]),
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.EnemyID]),
	)
	return title, body, nil
}

type acceptedSurrender struct {
	baseRenderer
}

func (n acceptedSurrender) unmarshal(text string) (goesi.AcceptedSurrender, set.Set[int64], error) {
	var data goesi.AcceptedSurrender
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.EntityID, data.OfferingID), nil
}

func (n acceptedSurrender) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n acceptedSurrender) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s's surrender accepted", entities[data.OfferingID].Name)
	body := fmt.Sprintf(
		"%s has accepted the surrender of %s in the war against %s, for **%s** ISK.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.OfferingID]),
		makeInfoLink(entities[data.EntityID]),
		humanize.Commaf(data.IskValue),
	)
	return title, body, nil
}

type allianceCapitalChanged struct {
	baseRenderer
}

func (n allianceCapitalChanged) unmarshal(text string) (goesi.AllianceCapitalChanged, set.Set[int64], error) {
	var data goesi.AllianceCapitalChanged
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AllianceID), nil
}

func (n allianceCapitalChanged) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allianceCapitalChanged) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has changed its capital", entities[data.AllianceID].Name)
	body := fmt.Sprintf(
		"%s has changed its capital system to %s.",
		makeInfoLink(entities[data.AllianceID]),
		makeInfoLink(solarSystem),
	)
	return title, body, nil
}

type allWarCorpJoinedAllianceMsg struct {
	baseRenderer
}

func (n allWarCorpJoinedAllianceMsg) unmarshal(text string) (goesi.AllWarCorpJoinedAllianceMsg, set.Set[int64], error) {
	var data goesi.AllWarCorpJoinedAllianceMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AllianceID, data.CorpID), nil
}

func (n allWarCorpJoinedAllianceMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allWarCorpJoinedAllianceMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has joined %s", entities[data.CorpID].Name, entities[data.AllianceID].Name)
	body := fmt.Sprintf(
		"%s has joined %s, inheriting its wars.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.AllianceID]),
	)
	return title, body, nil
}

func warDeclaredRender(ctx context.Context, verb string, againstID, declaredByID int64, cost float64, eus EVEUniverse) (string, string, error) {
	entities, err := eus.ToEntities(ctx, set.Of(againstID, declaredByID))
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s %s war against %s", entities[declaredByID].Name, verb, entities[againstID].Name)
	body := fmt.Sprintf(
		"%s has %s war against %s for **%s** ISK.",
		makeInfoLink(entities[declaredByID]),
		verb,
		makeInfoLink(entities[againstID]),
		humanize.Commaf(cost),
	)
	return title, body, nil
}

type allWarDeclaredMsg struct {
	baseRenderer
}

func (n allWarDeclaredMsg) unmarshal(text string) (goesi.AllWarDeclaredMsg, set.Set[int64], error) {
	var data goesi.AllWarDeclaredMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n allWarDeclaredMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allWarDeclaredMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, _, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	return warDeclaredRender(ctx, "declared", data.AgainstID, data.DeclaredByID, data.Cost, n.eus)
}

type allWarInvalidatedMsg struct {
	baseRenderer
}

func (n allWarInvalidatedMsg) unmarshal(text string) (goesi.AllWarInvalidatedMsg, set.Set[int64], error) {
	var data goesi.AllWarInvalidatedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n allWarInvalidatedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allWarInvalidatedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, _, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	return warDeclaredRender(ctx, "invalidated", data.AgainstID, data.DeclaredByID, data.Cost, n.eus)
}

type allWarRetractedMsg struct {
	baseRenderer
}

func (n allWarRetractedMsg) unmarshal(text string) (goesi.AllWarRetractedMsg, set.Set[int64], error) {
	var data goesi.AllWarRetractedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n allWarRetractedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allWarRetractedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, _, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	return warDeclaredRender(ctx, "retracted", data.AgainstID, data.DeclaredByID, data.Cost, n.eus)
}

type corpWarDeclaredMsg struct {
	baseRenderer
}

func (n corpWarDeclaredMsg) unmarshal(text string) (goesi.CorpWarDeclaredMsg, set.Set[int64], error) {
	var data goesi.CorpWarDeclaredMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n corpWarDeclaredMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpWarDeclaredMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, _, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	return warDeclaredRender(ctx, "declared", data.AgainstID, data.DeclaredByID, data.Cost, n.eus)
}

type corpWarFightingLegalMsg struct {
	baseRenderer
}

func (n corpWarFightingLegalMsg) unmarshal(text string) (goesi.CorpWarFightingLegalMsg, set.Set[int64], error) {
	var data goesi.CorpWarFightingLegalMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n corpWarFightingLegalMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpWarFightingLegalMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("War between %s and %s is now legal", entities[data.DeclaredByID].Name, entities[data.AgainstID].Name)
	body := fmt.Sprintf(
		"Fighting can now legally occur between %s and %s.",
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type corpWarInvalidatedMsg struct {
	baseRenderer
}

func (n corpWarInvalidatedMsg) unmarshal(text string) (goesi.CorpWarInvalidatedMsg, set.Set[int64], error) {
	var data goesi.CorpWarInvalidatedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n corpWarInvalidatedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpWarInvalidatedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, _, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	return warDeclaredRender(ctx, "invalidated", data.AgainstID, data.DeclaredByID, data.Cost, n.eus)
}

type corpWarRetractedMsg struct {
	baseRenderer
}

func (n corpWarRetractedMsg) unmarshal(text string) (goesi.CorpWarRetractedMsg, set.Set[int64], error) {
	var data goesi.CorpWarRetractedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n corpWarRetractedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpWarRetractedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, _, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	return warDeclaredRender(ctx, "retracted", data.AgainstID, data.DeclaredByID, data.Cost, n.eus)
}

type allyContractCancelled struct {
	baseRenderer
}

func (n allyContractCancelled) unmarshal(text string) (goesi.AllyContractCancelled, set.Set[int64], error) {
	var data goesi.AllyContractCancelled
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AggressorID, data.DefenderID), nil
}

func (n allyContractCancelled) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allyContractCancelled) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Ally contract cancelled"
	body := fmt.Sprintf(
		"The ally contract between %s and %s has been cancelled.",
		makeInfoLink(entities[data.AggressorID]),
		makeInfoLink(entities[data.DefenderID]),
	)
	return title, body, nil
}

type allyJoinedWarAggressorMsg struct {
	baseRenderer
}

func (n allyJoinedWarAggressorMsg) unmarshal(text string) (goesi.AllyJoinedWarAggressorMsg, set.Set[int64], error) {
	var data goesi.AllyJoinedWarAggressorMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AllyID, data.DefenderID), nil
}

func (n allyJoinedWarAggressorMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allyJoinedWarAggressorMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has joined the war as an aggressor", entities[data.AllyID].Name)
	body := fmt.Sprintf(
		"%s has joined the war against %s as an ally of the aggressor.",
		makeInfoLink(entities[data.AllyID]),
		makeInfoLink(entities[data.DefenderID]),
	)
	return title, body, nil
}

type allyJoinedWarAllyMsg struct {
	baseRenderer
}

func (n allyJoinedWarAllyMsg) unmarshal(text string) (goesi.AllyJoinedWarAllyMsg, set.Set[int64], error) {
	var data goesi.AllyJoinedWarAllyMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AggressorID, data.AllyID, data.DefenderID), nil
}

func (n allyJoinedWarAllyMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allyJoinedWarAllyMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has joined the war as your ally", entities[data.AllyID].Name)
	body := fmt.Sprintf(
		"%s has joined the war between %s and %s as your ally.",
		makeInfoLink(entities[data.AllyID]),
		makeInfoLink(entities[data.AggressorID]),
		makeInfoLink(entities[data.DefenderID]),
	)
	return title, body, nil
}

type allyJoinedWarDefenderMsg struct {
	baseRenderer
}

func (n allyJoinedWarDefenderMsg) unmarshal(text string) (goesi.AllyJoinedWarDefenderMsg, set.Set[int64], error) {
	var data goesi.AllyJoinedWarDefenderMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AggressorID, data.AllyID), nil
}

func (n allyJoinedWarDefenderMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allyJoinedWarDefenderMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has joined the war as a defender", entities[data.AllyID].Name)
	body := fmt.Sprintf(
		"%s has joined the war against %s as an ally of the defender.",
		makeInfoLink(entities[data.AllyID]),
		makeInfoLink(entities[data.AggressorID]),
	)
	return title, body, nil
}

type madeWarMutual struct {
	baseRenderer
}

func (n madeWarMutual) unmarshal(text string) (goesi.MadeWarMutual, set.Set[int64], error) {
	var data goesi.MadeWarMutual
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.EnemyID), nil
}

func (n madeWarMutual) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n madeWarMutual) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("War with %s is now mutual", entities[data.EnemyID].Name)
	body := fmt.Sprintf(
		"%s has made the war against %s mutual.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.EnemyID]),
	)
	return title, body, nil
}

type offeredSurrender struct {
	baseRenderer
}

func (n offeredSurrender) unmarshal(text string) (goesi.OfferedSurrender, set.Set[int64], error) {
	var data goesi.OfferedSurrender
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.EntityID, data.OfferedID), nil
}

func (n offeredSurrender) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n offeredSurrender) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has offered to surrender", entities[data.OfferedID].Name)
	body := fmt.Sprintf(
		"%s has offered the surrender of %s in the war against %s, for **%s** ISK.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.OfferedID]),
		makeInfoLink(entities[data.EntityID]),
		humanize.Commaf(data.IskValue),
	)
	return title, body, nil
}

type offeredToAlly struct {
	baseRenderer
}

func (n offeredToAlly) unmarshal(text string) (goesi.OfferedToAlly, set.Set[int64], error) {
	var data goesi.OfferedToAlly
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.DefenderID, data.EnemyID), nil
}

func (n offeredToAlly) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n offeredToAlly) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has offered an ally contract", entities[data.CharID].Name)
	body := fmt.Sprintf(
		"%s has offered %s an ally contract worth **%s** ISK in the war against %s.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.DefenderID]),
		humanize.Commaf(data.IskValue),
		makeInfoLink(entities[data.EnemyID]),
	)
	return title, body, nil
}

type retractsWar struct {
	baseRenderer
}

func (n retractsWar) unmarshal(text string) (goesi.RetractsWar, set.Set[int64], error) {
	var data goesi.RetractsWar
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.EnemyID), nil
}

func (n retractsWar) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n retractsWar) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("War against %s retracted", entities[data.EnemyID].Name)
	body := fmt.Sprintf(
		"%s has retracted the war against %s.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.EnemyID]),
	)
	return title, body, nil
}

type warAllyOfferDeclinedMsg struct {
	baseRenderer
}

func (n warAllyOfferDeclinedMsg) unmarshal(text string) (goesi.WarAllyOfferDeclinedMsg, set.Set[int64], error) {
	var data goesi.WarAllyOfferDeclinedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AggressorID, data.AllyID, data.CharID, data.DefenderID), nil
}

func (n warAllyOfferDeclinedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n warAllyOfferDeclinedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has declined an ally offer", entities[data.AllyID].Name)
	body := fmt.Sprintf(
		"%s has declined %s's offer to join the war between %s and %s as an ally.",
		makeInfoLink(entities[data.AllyID]),
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.AggressorID]),
		makeInfoLink(entities[data.DefenderID]),
	)
	return title, body, nil
}

type warSurrenderDeclinedMsg struct {
	baseRenderer
}

func (n warSurrenderDeclinedMsg) unmarshal(text string) (goesi.WarSurrenderDeclinedMsg, set.Set[int64], error) {
	var data goesi.WarSurrenderDeclinedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.OwnerID), nil
}

func (n warSurrenderDeclinedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n warSurrenderDeclinedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has declined a surrender offer", entities[data.OwnerID].Name)
	body := fmt.Sprintf(
		"%s has declined a surrender offer worth **%s** ISK.",
		makeInfoLink(entities[data.OwnerID]),
		humanize.Commaf(data.IskValue),
	)
	return title, body, nil
}

type warSurrenderOfferMsg struct {
	baseRenderer
}

func (n warSurrenderOfferMsg) unmarshal(text string) (goesi.WarSurrenderOfferMsg, set.Set[int64], error) {
	var data goesi.WarSurrenderOfferMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.OwnerID1, data.OwnerID2), nil
}

func (n warSurrenderOfferMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n warSurrenderOfferMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Surrender offer between %s and %s", entities[data.OwnerID1].Name, entities[data.OwnerID2].Name)
	body := fmt.Sprintf(
		"A surrender offer worth **%s** ISK has been made between %s and %s.",
		humanize.Commaf(data.IskValue),
		makeInfoLink(entities[data.OwnerID1]),
		makeInfoLink(entities[data.OwnerID2]),
	)
	return title, body, nil
}

type mercOfferedNegotiationMsg struct {
	baseRenderer
}

func (n mercOfferedNegotiationMsg) unmarshal(text string) (goesi.MercOfferedNegotiationMsg, set.Set[int64], error) {
	var data goesi.MercOfferedNegotiationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AggressorID, data.DefenderID, data.MercID), nil
}

func (n mercOfferedNegotiationMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n mercOfferedNegotiationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has offered mercenary negotiation", entities[data.MercID].Name)
	body := fmt.Sprintf(
		"%s has offered %s negotiation services worth **%s** ISK in the war against %s.",
		makeInfoLink(entities[data.MercID]),
		makeInfoLink(entities[data.DefenderID]),
		humanize.Commaf(data.IskValue),
		makeInfoLink(entities[data.AggressorID]),
	)
	return title, body, nil
}

type mercOfferRetractedMsg struct {
	baseRenderer
}

func (n mercOfferRetractedMsg) unmarshal(text string) (notification2.MercOfferRetractedMsg, set.Set[int64], error) {
	var data notification2.MercOfferRetractedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AggressorID, data.DefenderID, data.MercID), nil
}

func (n mercOfferRetractedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n mercOfferRetractedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has retracted their mercenary offer", entities[data.MercID].Name)
	body := fmt.Sprintf(
		"%s has retracted their offer to %s for negotiation services in the war against %s.",
		makeInfoLink(entities[data.MercID]),
		makeInfoLink(entities[data.DefenderID]),
		makeInfoLink(entities[data.AggressorID]),
	)
	return title, body, nil
}

func warDeclaredV2Render(ctx context.Context, verb string, againstID, declaredByID, delayHours int64, cost float64, eus EVEUniverse) (string, string, error) {
	entities, err := eus.ToEntities(ctx, set.Of(againstID, declaredByID))
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s %s war against %s", entities[declaredByID].Name, verb, entities[againstID].Name)
	body := fmt.Sprintf(
		"%s has %s war against %s for **%s** ISK.\n\n"+
			"Within **%d** hours fighting can legally occur between those involved.",
		makeInfoLink(entities[declaredByID]),
		verb,
		makeInfoLink(entities[againstID]),
		humanize.Commaf(cost),
		delayHours,
	)
	return title, body, nil
}

type allianceWarDeclaredV2 struct {
	baseRenderer
}

func (n allianceWarDeclaredV2) unmarshal(text string) (notification2.AllianceWarDeclaredV2, set.Set[int64], error) {
	var data notification2.AllianceWarDeclaredV2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n allianceWarDeclaredV2) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n allianceWarDeclaredV2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, _, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	return warDeclaredV2Render(ctx, "declared", data.AgainstID, data.DeclaredByID, data.DelayHours, data.Cost, n.eus)
}

type corpWarDeclaredV2 struct {
	baseRenderer
}

func (n corpWarDeclaredV2) unmarshal(text string) (notification2.CorpWarDeclaredV2, set.Set[int64], error) {
	var data notification2.CorpWarDeclaredV2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n corpWarDeclaredV2) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpWarDeclaredV2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, _, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	return warDeclaredV2Render(ctx, "declared", data.AgainstID, data.DeclaredByID, data.DelayHours, data.Cost, n.eus)
}

type warConcordInvalidates struct {
	baseRenderer
}

func (n warConcordInvalidates) unmarshal(text string) (notification2.WarConcordInvalidates, set.Set[int64], error) {
	var data notification2.WarConcordInvalidates
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n warConcordInvalidates) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n warConcordInvalidates) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "CONCORD invalidates war"
	body := fmt.Sprintf(
		"The war between %s and %s has been invalidated by CONCORD, "+
			"because at least one of the involved parties has become ineligible for war declarations.",
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type warRetracted struct {
	baseRenderer
}

func (n warRetracted) unmarshal(text string) (notification2.WarRetracted, set.Set[int64], error) {
	var data notification2.WarRetracted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AgainstID, data.DeclaredByID), nil
}

func (n warRetracted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n warRetracted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("War against %s retracted", entities[data.AgainstID].Name)
	body := fmt.Sprintf(
		"%s has retracted the war against %s.",
		makeInfoLink(entities[data.DeclaredByID]),
		makeInfoLink(entities[data.AgainstID]),
	)
	return title, body, nil
}
