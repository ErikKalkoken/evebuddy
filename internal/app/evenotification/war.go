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
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		data.DelayHours,
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
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
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
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
		makeEveEntityProfileLink(entities[data.EntityID]),
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
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
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
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
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
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
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
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		fromLDAPTime(data.EndDate).Format(app.DateTimeFormat),
	)
	body = out
	return title, body, nil
}

type acceptedAlly struct {
	baseRenderer
}

func (n acceptedAlly) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n acceptedAlly) unmarshal(text string) (goesi.AcceptedAlly, set.Set[int64], error) {
	var data goesi.AcceptedAlly
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllyID, data.CharID, data.EnemyID)
	return data, ids, nil
}

func (n acceptedAlly) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has accepted an ally contract", entities[data.AllyID].Name)
	body = fmt.Sprintf(
		"%s has accepted the ally contract in the war against %s, contracted by %s.",
		makeEveEntityProfileLink(entities[data.AllyID]),
		makeEveEntityProfileLink(entities[data.EnemyID]),
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}

type acceptedSurrender struct {
	baseRenderer
}

func (n acceptedSurrender) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n acceptedSurrender) unmarshal(text string) (goesi.AcceptedSurrender, set.Set[int64], error) {
	var data goesi.AcceptedSurrender
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.EntityID)
	return data, ids, nil
}

func (n acceptedSurrender) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Surrender accepted from %s", entities[data.EntityID].Name)
	body = fmt.Sprintf(
		"%s has accepted a surrender offer from %s for **%s** ISK.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.EntityID]),
		humanize.Commaf(data.IskValue),
	)
	return title, body, nil
}

type allWarCorpJoinedAllianceMsg struct {
	baseRenderer
}

func (n allWarCorpJoinedAllianceMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allWarCorpJoinedAllianceMsg) unmarshal(text string) (goesi.AllWarCorpJoinedAllianceMsg, set.Set[int64], error) {
	var data goesi.AllWarCorpJoinedAllianceMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllianceID, data.CorpID)
	return data, ids, nil
}

func (n allWarCorpJoinedAllianceMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has joined %s", entities[data.CorpID].Name, entities[data.AllianceID].Name)
	body = fmt.Sprintf(
		"%s has joined %s during an ongoing war. War rights are being inherited.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.AllianceID]),
	)
	return title, body, nil
}

type allWarDeclaredMsg struct {
	baseRenderer
}

func (n allWarDeclaredMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allWarDeclaredMsg) unmarshal(text string) (goesi.AllWarDeclaredMsg, set.Set[int64], error) {
	var data goesi.AllWarDeclaredMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n allWarDeclaredMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s declares war against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body = fmt.Sprintf(
		"%s has declared war on %s. "+
			"Within **%d** hours fighting can legally occur between those involved.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		data.DelayHours,
	)
	return title, body, nil
}

type allWarInvalidatedMsg struct {
	baseRenderer
}

func (n allWarInvalidatedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allWarInvalidatedMsg) unmarshal(text string) (goesi.AllWarInvalidatedMsg, set.Set[int64], error) {
	var data goesi.AllWarInvalidatedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n allWarInvalidatedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("War declared by %s against %s has been invalidated",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body = fmt.Sprintf(
		"The war between %s and %s has been invalidated by CONCORD, "+
			"because at least one of the involved parties has become ineligible for war declarations.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type allWarRetractedMsg struct {
	baseRenderer
}

func (n allWarRetractedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allWarRetractedMsg) unmarshal(text string) (goesi.AllWarRetractedMsg, set.Set[int64], error) {
	var data goesi.AllWarRetractedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n allWarRetractedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s retracts war against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body = fmt.Sprintf(
		"%s has retracted the war against %s.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type allyContractCancelled struct {
	baseRenderer
}

func (n allyContractCancelled) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allyContractCancelled) unmarshal(text string) (goesi.AllyContractCancelled, set.Set[int64], error) {
	var data goesi.AllyContractCancelled
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorID, data.DefenderID)
	return data, ids, nil
}

func (n allyContractCancelled) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Ally contract cancelled"
	body = fmt.Sprintf(
		"The ally contract in the war between %s and %s has been cancelled. "+
			"It was due to end at **%s**.",
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
		fromLDAPTime(data.TimeFinished).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type allyJoinedWarAggressorMsg struct {
	baseRenderer
}

func (n allyJoinedWarAggressorMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allyJoinedWarAggressorMsg) unmarshal(text string) (goesi.AllyJoinedWarAggressorMsg, set.Set[int64], error) {
	var data goesi.AllyJoinedWarAggressorMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllyID, data.DefenderID)
	return data, ids, nil
}

func (n allyJoinedWarAggressorMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has joined the war as your ally", entities[data.AllyID].Name)
	body = fmt.Sprintf(
		"%s has joined the war on your side as an ally against %s. "+
			"The ally contract starts at **%s**.",
		makeEveEntityProfileLink(entities[data.AllyID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
		fromLDAPTime(data.StartTime).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type allyJoinedWarAllyMsg struct {
	baseRenderer
}

func (n allyJoinedWarAllyMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allyJoinedWarAllyMsg) unmarshal(text string) (goesi.AllyJoinedWarAllyMsg, set.Set[int64], error) {
	var data goesi.AllyJoinedWarAllyMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorID, data.AllyID, data.DefenderID)
	return data, ids, nil
}

func (n allyJoinedWarAllyMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has joined the war as an ally", entities[data.AllyID].Name)
	body = fmt.Sprintf(
		"%s has joined the war between %s and %s as an ally to the aggressor. "+
			"The ally contract starts at **%s**.",
		makeEveEntityProfileLink(entities[data.AllyID]),
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
		fromLDAPTime(data.StartTime).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type allyJoinedWarDefenderMsg struct {
	baseRenderer
}

func (n allyJoinedWarDefenderMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allyJoinedWarDefenderMsg) unmarshal(text string) (goesi.AllyJoinedWarDefenderMsg, set.Set[int64], error) {
	var data goesi.AllyJoinedWarDefenderMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorID, data.AllyID)
	return data, ids, nil
}

func (n allyJoinedWarDefenderMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has joined the war against you", entities[data.AllyID].Name)
	body = fmt.Sprintf(
		"%s has joined the war as an ally to %s. "+
			"The ally contract starts at **%s**.",
		makeEveEntityProfileLink(entities[data.AllyID]),
		makeEveEntityProfileLink(entities[data.AggressorID]),
		fromLDAPTime(data.StartTime).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type corpWarDeclaredMsg struct {
	baseRenderer
}

func (n corpWarDeclaredMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpWarDeclaredMsg) unmarshal(text string) (goesi.CorpWarDeclaredMsg, set.Set[int64], error) {
	var data goesi.CorpWarDeclaredMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n corpWarDeclaredMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s declares war against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body = fmt.Sprintf(
		"%s has declared war on %s. "+
			"Within **24** hours fighting can legally occur between those involved.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type corpWarFightingLegalMsg struct {
	baseRenderer
}

func (n corpWarFightingLegalMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpWarFightingLegalMsg) unmarshal(text string) (goesi.CorpWarFightingLegalMsg, set.Set[int64], error) {
	var data goesi.CorpWarFightingLegalMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n corpWarFightingLegalMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("War with %s is now active", entities[data.AgainstID].Name)
	body = fmt.Sprintf(
		"The war declared by %s against %s is now in effect. "+
			"Combat is now legal between the involved parties.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type corpWarInvalidatedMsg struct {
	baseRenderer
}

func (n corpWarInvalidatedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpWarInvalidatedMsg) unmarshal(text string) (goesi.CorpWarInvalidatedMsg, set.Set[int64], error) {
	var data goesi.CorpWarInvalidatedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n corpWarInvalidatedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("War between %s and %s has been invalidated",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body = fmt.Sprintf(
		"The war between %s and %s has been invalidated by CONCORD, "+
			"because at least one of the involved parties has become ineligible for war declarations.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type corpWarRetractedMsg struct {
	baseRenderer
}

func (n corpWarRetractedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpWarRetractedMsg) unmarshal(text string) (goesi.CorpWarRetractedMsg, set.Set[int64], error) {
	var data goesi.CorpWarRetractedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n corpWarRetractedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s retracts war against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body = fmt.Sprintf(
		"%s has retracted the war against %s.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type madeWarMutual struct {
	baseRenderer
}

func (n madeWarMutual) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n madeWarMutual) unmarshal(text string) (goesi.MadeWarMutual, set.Set[int64], error) {
	var data goesi.MadeWarMutual
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.EnemyID)
	return data, ids, nil
}

func (n madeWarMutual) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("War with %s has been made mutual", entities[data.EnemyID].Name)
	body = fmt.Sprintf(
		"%s has made the war with %s mutual.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.EnemyID]),
	)
	return title, body, nil
}

type mercOfferedNegotiationMsg struct {
	baseRenderer
}

func (n mercOfferedNegotiationMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n mercOfferedNegotiationMsg) unmarshal(text string) (goesi.MercOfferedNegotiationMsg, set.Set[int64], error) {
	var data goesi.MercOfferedNegotiationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorID, data.DefenderID, data.MercID)
	return data, ids, nil
}

func (n mercOfferedNegotiationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Mercenary offered to negotiate end of war"
	body = fmt.Sprintf(
		"%s has offered to negotiate an end to the war between %s and %s for **%s** ISK.",
		makeEveEntityProfileLink(entities[data.MercID]),
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
		humanize.Commaf(data.IskValue),
	)
	return title, body, nil
}

type mercOfferRetractedMsg struct {
	baseRenderer
}

func (n mercOfferRetractedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n mercOfferRetractedMsg) unmarshal(text string) (notification2.MercOfferRetractedMsg, set.Set[int64], error) {
	var data notification2.MercOfferRetractedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorID, data.DefenderID, data.MercID)
	return data, ids, nil
}

func (n mercOfferRetractedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Mercenary retracted negotiation offer"
	body = fmt.Sprintf(
		"%s has retracted their negotiation offer in the war between %s and %s.",
		makeEveEntityProfileLink(entities[data.MercID]),
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
	)
	return title, body, nil
}

type mutualWarExpired struct {
	baseRenderer
}

func (n mutualWarExpired) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n mutualWarExpired) unmarshal(text string) (notification2.MutualWarExpired, set.Set[int64], error) {
	var data notification2.MutualWarExpired
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n mutualWarExpired) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Mutual war has expired"
	body = fmt.Sprintf(
		"The mutual war between %s and %s has expired.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type mutualWarInviteAccepted struct {
	baseRenderer
}

func (n mutualWarInviteAccepted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n mutualWarInviteAccepted) unmarshal(text string) (notification2.MutualWarInviteAccepted, set.Set[int64], error) {
	var data notification2.MutualWarInviteAccepted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n mutualWarInviteAccepted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Mutual war invitation accepted"
	body = fmt.Sprintf(
		"%s has accepted the mutual war invitation from %s.",
		makeEveEntityProfileLink(entities[data.AgainstID]),
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
	)
	return title, body, nil
}

type mutualWarInviteRejected struct {
	baseRenderer
}

func (n mutualWarInviteRejected) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n mutualWarInviteRejected) unmarshal(text string) (notification2.MutualWarInviteRejected, set.Set[int64], error) {
	var data notification2.MutualWarInviteRejected
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n mutualWarInviteRejected) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Mutual war invitation rejected"
	body = fmt.Sprintf(
		"%s has rejected the mutual war invitation from %s.",
		makeEveEntityProfileLink(entities[data.AgainstID]),
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
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
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Mutual war invitation sent to %s", entities[data.AgainstID].Name)
	body = fmt.Sprintf(
		"%s has sent a mutual war invitation to %s.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type offeredSurrender struct {
	baseRenderer
}

func (n offeredSurrender) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n offeredSurrender) unmarshal(text string) (goesi.OfferedSurrender, set.Set[int64], error) {
	var data goesi.OfferedSurrender
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.EntityID)
	return data, ids, nil
}

func (n offeredSurrender) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has offered surrender", entities[data.EntityID].Name)
	body = fmt.Sprintf(
		"%s acting on behalf of %s has offered a surrender for **%s** ISK.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.EntityID]),
		humanize.Commaf(data.IskValue),
	)
	return title, body, nil
}

type offeredToAlly struct {
	baseRenderer
}

func (n offeredToAlly) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n offeredToAlly) unmarshal(text string) (goesi.OfferedToAlly, set.Set[int64], error) {
	var data goesi.OfferedToAlly
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.DefenderID, data.EnemyID)
	return data, ids, nil
}

func (n offeredToAlly) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "War ally contract offered"
	body = fmt.Sprintf(
		"%s has offered an ally contract for **%s** ISK to join the war against %s in support of %s.",
		makeEveEntityProfileLink(entities[data.CharID]),
		humanize.Commaf(data.IskValue),
		makeEveEntityProfileLink(entities[data.EnemyID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
	)
	return title, body, nil
}

type offerToAllyRetracted struct {
	baseRenderer
}

func (n offerToAllyRetracted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n offerToAllyRetracted) unmarshal(text string) (notification2.OfferToAllyRetracted, set.Set[int64], error) {
	var data notification2.OfferToAllyRetracted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorID, data.DefenderID, data.EnemyID)
	return data, ids, nil
}

func (n offerToAllyRetracted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "War ally offer retracted"
	body = fmt.Sprintf(
		"The ally contract offer in the war between %s and %s has been retracted by %s.",
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.EnemyID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
	)
	return title, body, nil
}

type retractsWar struct {
	baseRenderer
}

func (n retractsWar) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n retractsWar) unmarshal(text string) (goesi.RetractsWar, set.Set[int64], error) {
	var data goesi.RetractsWar
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.EnemyID)
	return data, ids, nil
}

func (n retractsWar) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s retracts war against %s",
		entities[data.CharID].Name,
		entities[data.EnemyID].Name,
	)
	body = fmt.Sprintf(
		"%s has retracted the war against %s.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.EnemyID]),
	)
	return title, body, nil
}

type warAllyOfferDeclinedMsg struct {
	baseRenderer
}

func (n warAllyOfferDeclinedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warAllyOfferDeclinedMsg) unmarshal(text string) (goesi.WarAllyOfferDeclinedMsg, set.Set[int64], error) {
	var data goesi.WarAllyOfferDeclinedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorID, data.AllyID, data.DefenderID)
	return data, ids, nil
}

func (n warAllyOfferDeclinedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s declined ally offer", entities[data.AllyID].Name)
	body = fmt.Sprintf(
		"%s has declined the ally contract offer in the war between %s and %s.",
		makeEveEntityProfileLink(entities[data.AllyID]),
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.DefenderID]),
	)
	return title, body, nil
}

type warAllyInherited struct {
	baseRenderer
}

func (n warAllyInherited) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warAllyInherited) unmarshal(text string) (notification2.WarAllyInherited, set.Set[int64], error) {
	var data notification2.WarAllyInherited
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.AllyID)
	return data, ids, nil
}

func (n warAllyInherited) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s inherited war ally status", entities[data.AllyID].Name)
	body = fmt.Sprintf(
		"%s has inherited the war ally status against %s.",
		makeEveEntityProfileLink(entities[data.AllyID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type warConcordInvalidates struct {
	baseRenderer
}

func (n warConcordInvalidates) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warConcordInvalidates) unmarshal(text string) (notification2.WarConcordInvalidates, set.Set[int64], error) {
	var data notification2.WarConcordInvalidates
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warConcordInvalidates) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	body = fmt.Sprintf(
		"CONCORD has invalidated the war between %s and %s, "+
			"because at least one of the involved parties has become ineligible for war declarations.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type warEndedHqSecurityDrop struct {
	baseRenderer
}

func (n warEndedHqSecurityDrop) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warEndedHqSecurityDrop) unmarshal(text string) (notification2.WarEndedHqSecurityDrop, set.Set[int64], error) {
	var data notification2.WarEndedHqSecurityDrop
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warEndedHqSecurityDrop) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "War ended due to HQ security drop"
	body = fmt.Sprintf(
		"The war between %s and %s has ended because "+
			"the war HQ dropped below the required security status.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}

type warRetracted struct {
	baseRenderer
}

func (n warRetracted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warRetracted) unmarshal(text string) (notification2.WarRetracted, set.Set[int64], error) {
	var data notification2.WarRetracted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warRetracted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s retracts war against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body = fmt.Sprintf(
		"%s has retracted the war against %s.\n\n"+
			"After **%s** CONCORD will again respond to any hostilities between those involved with full force.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		fromLDAPTime(data.EndDate).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type warSurrenderDeclinedMsg struct {
	baseRenderer
}

func (n warSurrenderDeclinedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warSurrenderDeclinedMsg) unmarshal(text string) (goesi.WarSurrenderDeclinedMsg, set.Set[int64], error) {
	var data goesi.WarSurrenderDeclinedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.OwnerID)
	return data, ids, nil
}

func (n warSurrenderDeclinedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Surrender offer declined"
	body = fmt.Sprintf(
		"%s has declined the surrender offer for **%s** ISK.",
		makeEveEntityProfileLink(entities[data.OwnerID]),
		humanize.Commaf(data.IskValue),
	)
	return title, body, nil
}

type warSurrenderOfferMsg struct {
	baseRenderer
}

func (n warSurrenderOfferMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n warSurrenderOfferMsg) unmarshal(text string) (goesi.WarSurrenderOfferMsg, set.Set[int64], error) {
	var data goesi.WarSurrenderOfferMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.OwnerID1, data.OwnerID2)
	return data, ids, nil
}

func (n warSurrenderOfferMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "War surrender offer"
	body = fmt.Sprintf(
		"%s has offered a surrender to %s for **%s** ISK.",
		makeEveEntityProfileLink(entities[data.OwnerID1]),
		makeEveEntityProfileLink(entities[data.OwnerID2]),
		humanize.Commaf(data.IskValue),
	)
	return title, body, nil
}

type corpBecameWarEligible struct {
	baseRenderer
}

func (n corpBecameWarEligible) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Corporation is now war eligible"
	body := "Your corporation has accumulated enough assets to become eligible for war declarations."
	return title, body, nil
}

type corpNoLongerWarEligible struct {
	baseRenderer
}

func (n corpNoLongerWarEligible) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Corporation is no longer war eligible"
	body := "Your corporation no longer has sufficient assets to remain eligible for war declarations."
	return title, body, nil
}

type allianceWarDeclaredV2 struct {
	baseRenderer
}

func (n allianceWarDeclaredV2) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allianceWarDeclaredV2) unmarshal(text string) (notification2.AllianceWarDeclaredV2, set.Set[int64], error) {
	var data notification2.AllianceWarDeclaredV2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n allianceWarDeclaredV2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s declares war against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body = fmt.Sprintf(
		"%s has declared war on %s. "+
			"Within **%d** hours fighting can legally occur between those involved.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
		data.DelayHours,
	)
	return title, body, nil
}

type corpWarDeclaredV2 struct {
	baseRenderer
}

func (n corpWarDeclaredV2) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpWarDeclaredV2) unmarshal(text string) (notification2.CorpWarDeclaredV2, set.Set[int64], error) {
	var data notification2.CorpWarDeclaredV2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n corpWarDeclaredV2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s declares war against %s",
		entities[data.DeclaredByID].Name,
		entities[data.AgainstID].Name,
	)
	body = fmt.Sprintf(
		"%s has declared war on %s. "+
			"Within **24** hours fighting can legally occur between those involved.",
		makeEveEntityProfileLink(entities[data.DeclaredByID]),
		makeEveEntityProfileLink(entities[data.AgainstID]),
	)
	return title, body, nil
}
