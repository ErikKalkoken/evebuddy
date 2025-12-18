package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/antihax/goesi/notification"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type allWarSurrenderMsg struct {
	baseRenderer
}

func (n allWarSurrenderMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n allWarSurrenderMsg) unmarshal(text string) (notification.AllWarSurrenderMsg, setInt32, error) {
	var data notification.AllWarSurrenderMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n allWarSurrenderMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n corpWarSurrenderMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n corpWarSurrenderMsg) unmarshal(text string) (notification.CorpWarSurrenderMsg, setInt32, error) {
	var data notification.CorpWarSurrenderMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n corpWarSurrenderMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n declareWar) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n declareWar) unmarshal(text string) (notification.DeclareWar, setInt32, error) {
	var data notification.DeclareWar
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID, data.DefenderID, data.EntityID)
	return data, ids, nil
}

func (n declareWar) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n warAdopted) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n warAdopted) unmarshal(text string) (notification.WarAdopted, setInt32, error) {
	var data notification.WarAdopted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID, data.AllianceID)
	return data, ids, nil
}

func (n warAdopted) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n warDeclared) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n warDeclared) unmarshal(text string) (notification.WarDeclared, setInt32, error) {
	var data notification.WarDeclared
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warDeclared) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n warHQRemovedFromSpace) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n warHQRemovedFromSpace) unmarshal(text string) (notification.WarHQRemovedFromSpace, setInt32, error) {
	var data notification.WarHQRemovedFromSpace
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warHQRemovedFromSpace) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n warInherited) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n warInherited) unmarshal(text string) (notification.WarInherited, setInt32, error) {
	var data notification.WarInherited
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
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

func (n warInherited) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n warInvalid) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n warInvalid) unmarshal(text string) (notification.WarInvalid, setInt32, error) {
	var data notification.WarInvalid
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warInvalid) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n warRetractedByConcord) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n warRetractedByConcord) unmarshal(text string) (notification.WarRetractedByConcord, setInt32, error) {
	var data notification.WarRetractedByConcord
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AgainstID, data.DeclaredByID)
	return data, ids, nil
}

func (n warRetractedByConcord) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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
