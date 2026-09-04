package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"
)

type facWarCorpJoinRequestMsg struct {
	baseRenderer
}

func (n facWarCorpJoinRequestMsg) unmarshal(text string) (goesi.FacWarCorpJoinRequestMsg, set.Set[int64], error) {
	var data goesi.FacWarCorpJoinRequestMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.FactionID), nil
}

func (n facWarCorpJoinRequestMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n facWarCorpJoinRequestMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has applied to join %s", entities[data.CorpID].Name, entities[data.FactionID].Name)
	body := fmt.Sprintf(
		"%s has applied to join %s in the war effort.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type facWarCorpJoinWithdrawMsg struct {
	baseRenderer
}

func (n facWarCorpJoinWithdrawMsg) unmarshal(text string) (goesi.FacWarCorpJoinWithdrawMsg, set.Set[int64], error) {
	var data goesi.FacWarCorpJoinWithdrawMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.FactionID), nil
}

func (n facWarCorpJoinWithdrawMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n facWarCorpJoinWithdrawMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has withdrawn its application to join %s", entities[data.CorpID].Name, entities[data.FactionID].Name)
	body := fmt.Sprintf(
		"%s has withdrawn its application to join %s in the war effort.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type facWarCorpLeaveRequestMsg struct {
	baseRenderer
}

func (n facWarCorpLeaveRequestMsg) unmarshal(text string) (goesi.FacWarCorpLeaveRequestMsg, set.Set[int64], error) {
	var data goesi.FacWarCorpLeaveRequestMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.FactionID), nil
}

func (n facWarCorpLeaveRequestMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n facWarCorpLeaveRequestMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has applied to leave %s", entities[data.CorpID].Name, entities[data.FactionID].Name)
	body := fmt.Sprintf(
		"%s has applied to leave %s.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type facWarCorpLeaveWithdrawMsg struct {
	baseRenderer
}

func (n facWarCorpLeaveWithdrawMsg) unmarshal(text string) (goesi.FacWarCorpLeaveWithdrawMsg, set.Set[int64], error) {
	var data goesi.FacWarCorpLeaveWithdrawMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.FactionID), nil
}

func (n facWarCorpLeaveWithdrawMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n facWarCorpLeaveWithdrawMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has withdrawn its application to leave %s", entities[data.CorpID].Name, entities[data.FactionID].Name)
	body := fmt.Sprintf(
		"%s has withdrawn its application to leave %s.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type facWarLPDisqualifiedEvent struct {
	baseRenderer
}

func (n facWarLPDisqualifiedEvent) unmarshal(text string) (goesi.FacWarLPDisqualifiedEvent, set.Set[int64], error) {
	var data goesi.FacWarLPDisqualifiedEvent
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharRefID, data.CorpID), nil
}

func (n facWarLPDisqualifiedEvent) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n facWarLPDisqualifiedEvent) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Loyalty points disqualified"
	body := fmt.Sprintf(
		"%s of %s has had **%d** loyalty points disqualified for participating in an event.",
		makeInfoLink(entities[data.CharRefID]),
		makeInfoLink(entities[data.CorpID]),
		data.Amount,
	)
	return title, body, nil
}

type facWarLPDisqualifiedKill struct {
	baseRenderer
}

func (n facWarLPDisqualifiedKill) unmarshal(text string) (goesi.FacWarLPDisqualifiedKill, set.Set[int64], error) {
	var data goesi.FacWarLPDisqualifiedKill
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharRefID, data.CorpID), nil
}

func (n facWarLPDisqualifiedKill) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n facWarLPDisqualifiedKill) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Loyalty points disqualified"
	body := fmt.Sprintf(
		"%s of %s has had **%d** loyalty points disqualified for a kill.",
		makeInfoLink(entities[data.CharRefID]),
		makeInfoLink(entities[data.CorpID]),
		data.Amount,
	)
	return title, body, nil
}

type facWarLPPayoutEvent struct {
	baseRenderer
}

func (n facWarLPPayoutEvent) unmarshal(text string) (goesi.FacWarLPPayoutEvent, set.Set[int64], error) {
	var data goesi.FacWarLPPayoutEvent
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharRefID, data.CorpID), nil
}

func (n facWarLPPayoutEvent) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n facWarLPPayoutEvent) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Loyalty points awarded: %d", data.Amount)
	body := fmt.Sprintf(
		"%s of %s has been awarded **%d** loyalty points for participating in an event.",
		makeInfoLink(entities[data.CharRefID]),
		makeInfoLink(entities[data.CorpID]),
		data.Amount,
	)
	return title, body, nil
}

type facWarLPPayoutKill struct {
	baseRenderer
}

func (n facWarLPPayoutKill) unmarshal(text string) (goesi.FacWarLPPayoutKill, set.Set[int64], error) {
	var data goesi.FacWarLPPayoutKill
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharRefID, data.CorpID), nil
}

func (n facWarLPPayoutKill) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n facWarLPPayoutKill) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Loyalty points awarded: %d", data.Amount)
	body := fmt.Sprintf(
		"%s of %s has been awarded **%d** loyalty points for a kill.",
		makeInfoLink(entities[data.CharRefID]),
		makeInfoLink(entities[data.CorpID]),
		data.Amount,
	)
	return title, body, nil
}

type fwAllianceWarningMsg struct {
	baseRenderer
}

func (n fwAllianceWarningMsg) unmarshal(text string) (goesi.FWAllianceWarningMsg, set.Set[int64], error) {
	var data goesi.FWAllianceWarningMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.AllianceID, data.FactionID), nil
}

func (n fwAllianceWarningMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n fwAllianceWarningMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Faction warfare warning for %s", entities[data.AllianceID].Name)
	body := fmt.Sprintf(
		"%s's standing with %s is dropping below the required level of **%.1f**.\n\n"+
			"The following corporations are at risk of being kicked: %s.",
		makeInfoLink(entities[data.AllianceID]),
		makeInfoLink(entities[data.FactionID]),
		data.RequiredStanding,
		data.CorpList,
	)
	return title, body, nil
}

type fwCharRankGainMsg struct {
	baseRenderer
}

func (n fwCharRankGainMsg) unmarshal(text string) (goesi.FWCharRankGainMsg, set.Set[int64], error) {
	var data goesi.FWCharRankGainMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.FactionID), nil
}

func (n fwCharRankGainMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n fwCharRankGainMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Faction warfare rank increased to %d", data.NewRank)
	body := fmt.Sprintf(
		"Your faction warfare rank with %s has increased to **%d**.",
		makeInfoLink(entities[data.FactionID]),
		data.NewRank,
	)
	return title, body, nil
}

type fwCharRankLossMsg struct {
	baseRenderer
}

func (n fwCharRankLossMsg) unmarshal(text string) (goesi.FWCharRankLossMsg, set.Set[int64], error) {
	var data goesi.FWCharRankLossMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.FactionID), nil
}

func (n fwCharRankLossMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n fwCharRankLossMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Faction warfare rank decreased to %d", data.NewRank)
	body := fmt.Sprintf(
		"Your faction warfare rank with %s has decreased to **%d**.",
		makeInfoLink(entities[data.FactionID]),
		data.NewRank,
	)
	return title, body, nil
}

type fwCorpJoinMsg struct {
	baseRenderer
}

func (n fwCorpJoinMsg) unmarshal(text string) (goesi.FWCorpJoinMsg, set.Set[int64], error) {
	var data goesi.FWCorpJoinMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.FactionID), nil
}

func (n fwCorpJoinMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n fwCorpJoinMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has joined %s", entities[data.CorpID].Name, entities[data.FactionID].Name)
	body := fmt.Sprintf(
		"%s has joined %s in the war effort.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type fwCorpKickMsg struct {
	baseRenderer
}

func (n fwCorpKickMsg) unmarshal(text string) (goesi.FWCorpKickMsg, set.Set[int64], error) {
	var data goesi.FWCorpKickMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.FactionID), nil
}

func (n fwCorpKickMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n fwCorpKickMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has been kicked from %s", entities[data.CorpID].Name, entities[data.FactionID].Name)
	body := fmt.Sprintf(
		"%s has been kicked out of %s because its standing of **%.1f** "+
			"dropped below the required level of **%.1f**.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.FactionID]),
		data.CurrentStanding,
		data.RequiredStanding,
	)
	return title, body, nil
}

type fwCorpLeaveMsg struct {
	baseRenderer
}

func (n fwCorpLeaveMsg) unmarshal(text string) (goesi.FWCorpLeaveMsg, set.Set[int64], error) {
	var data goesi.FWCorpLeaveMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.FactionID), nil
}

func (n fwCorpLeaveMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n fwCorpLeaveMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has left %s", entities[data.CorpID].Name, entities[data.FactionID].Name)
	body := fmt.Sprintf(
		"%s has left %s.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type fwCorpWarningMsg struct {
	baseRenderer
}

func (n fwCorpWarningMsg) unmarshal(text string) (goesi.FWCorpWarningMsg, set.Set[int64], error) {
	var data goesi.FWCorpWarningMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.FactionID), nil
}

func (n fwCorpWarningMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n fwCorpWarningMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Faction warfare warning for %s", entities[data.CorpID].Name)
	body := fmt.Sprintf(
		"%s's standing with %s of **%.1f** is dropping below the required level of **%.1f**.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.FactionID]),
		data.CurrentStanding,
		data.RequiredStanding,
	)
	return title, body, nil
}
