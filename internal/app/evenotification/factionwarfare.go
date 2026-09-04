package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
)

type fwAllianceKickMsg struct {
	baseRenderer
}

func (n fwAllianceKickMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwAllianceKickMsg) unmarshal(text string) (notification2.FWAllianceKickMsg, set.Set[int64], error) {
	var data notification2.FWAllianceKickMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllianceID, data.FactionID)
	return data, ids, nil
}

func (n fwAllianceKickMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has been kicked from faction warfare", entities[data.AllianceID].Name)
	body = fmt.Sprintf(
		"%s has been kicked from faction warfare by %s due to insufficient standings.",
		makeEveEntityProfileLink(entities[data.AllianceID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type fwAllianceWarningMsg struct {
	baseRenderer
}

func (n fwAllianceWarningMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwAllianceWarningMsg) unmarshal(text string) (goesi.FWAllianceWarningMsg, set.Set[int64], error) {
	var data goesi.FWAllianceWarningMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllianceID, data.FactionID)
	return data, ids, nil
}

func (n fwAllianceWarningMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s standing warning from %s", entities[data.AllianceID].Name, entities[data.FactionID].Name)
	body = fmt.Sprintf(
		"%s has received a warning from %s. "+
			"Your current standing does not meet the required standing of **%.2f**. "+
			"Your alliance will be kicked from faction warfare if standings do not improve.",
		makeEveEntityProfileLink(entities[data.AllianceID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
		data.RequiredStanding,
	)
	return title, body, nil
}

type fwCharKickMsg struct {
	baseRenderer
}

func (n fwCharKickMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwCharKickMsg) unmarshal(text string) (notification2.FWCharKickMsg, set.Set[int64], error) {
	var data notification2.FWCharKickMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.FactionID)
	return data, ids, nil
}

func (n fwCharKickMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "You have been kicked from faction warfare"
	body = fmt.Sprintf(
		"You have been kicked from faction warfare by %s due to insufficient standings.",
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type fwCharRankGainMsg struct {
	baseRenderer
}

func (n fwCharRankGainMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwCharRankGainMsg) unmarshal(text string) (goesi.FWCharRankGainMsg, set.Set[int64], error) {
	var data goesi.FWCharRankGainMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.FactionID)
	return data, ids, nil
}

func (n fwCharRankGainMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Faction warfare rank gained: rank %d", data.NewRank)
	body = fmt.Sprintf(
		"You have been promoted to rank **%d** in %s faction warfare.",
		data.NewRank,
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type fwCharRankLossMsg struct {
	baseRenderer
}

func (n fwCharRankLossMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwCharRankLossMsg) unmarshal(text string) (goesi.FWCharRankLossMsg, set.Set[int64], error) {
	var data goesi.FWCharRankLossMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.FactionID)
	return data, ids, nil
}

func (n fwCharRankLossMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Faction warfare rank lost: now rank %d", data.NewRank)
	body = fmt.Sprintf(
		"You have been demoted to rank **%d** in %s faction warfare.",
		data.NewRank,
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type fwCharWarningMsg struct {
	baseRenderer
}

func (n fwCharWarningMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwCharWarningMsg) unmarshal(text string) (notification2.FWCharWarningMsg, set.Set[int64], error) {
	var data notification2.FWCharWarningMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.FactionID)
	return data, ids, nil
}

func (n fwCharWarningMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Faction warfare standing warning from %s", entities[data.FactionID].Name)
	body = fmt.Sprintf(
		"You have received a standing warning from %s. "+
			"Your current standing of **%.2f** does not meet the required standing of **%.2f**. "+
			"You will be kicked from faction warfare if standings do not improve.",
		makeEveEntityProfileLink(entities[data.FactionID]),
		data.CurrentStanding,
		data.RequiredStanding,
	)
	return title, body, nil
}

type fwCorpJoinMsg struct {
	baseRenderer
}

func (n fwCorpJoinMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwCorpJoinMsg) unmarshal(text string) (goesi.FWCorpJoinMsg, set.Set[int64], error) {
	var data goesi.FWCorpJoinMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.FactionID)
	return data, ids, nil
}

func (n fwCorpJoinMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has joined faction warfare", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has joined faction warfare on the side of %s.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type fwCorpKickMsg struct {
	baseRenderer
}

func (n fwCorpKickMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwCorpKickMsg) unmarshal(text string) (goesi.FWCorpKickMsg, set.Set[int64], error) {
	var data goesi.FWCorpKickMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.FactionID)
	return data, ids, nil
}

func (n fwCorpKickMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has been kicked from faction warfare", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has been kicked from faction warfare by %s. "+
			"Current standing: **%.2f**, required standing: **%.2f**.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
		data.CurrentStanding,
		data.RequiredStanding,
	)
	return title, body, nil
}

type fwCorpLeaveMsg struct {
	baseRenderer
}

func (n fwCorpLeaveMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwCorpLeaveMsg) unmarshal(text string) (goesi.FWCorpLeaveMsg, set.Set[int64], error) {
	var data goesi.FWCorpLeaveMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.FactionID)
	return data, ids, nil
}

func (n fwCorpLeaveMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has left faction warfare", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has left faction warfare and departed from %s.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type fwCorpWarningMsg struct {
	baseRenderer
}

func (n fwCorpWarningMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n fwCorpWarningMsg) unmarshal(text string) (goesi.FWCorpWarningMsg, set.Set[int64], error) {
	var data goesi.FWCorpWarningMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.FactionID)
	return data, ids, nil
}

func (n fwCorpWarningMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s standing warning from %s", entities[data.CorpID].Name, entities[data.FactionID].Name)
	body = fmt.Sprintf(
		"%s has received a standing warning from %s. "+
			"Current standing: **%.2f**, required standing: **%.2f**. "+
			"Your corporation will be kicked from faction warfare if standings do not improve.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
		data.CurrentStanding,
		data.RequiredStanding,
	)
	return title, body, nil
}

type facWarCorpJoinRequestMsg struct {
	baseRenderer
}

func (n facWarCorpJoinRequestMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n facWarCorpJoinRequestMsg) unmarshal(text string) (goesi.FacWarCorpJoinRequestMsg, set.Set[int64], error) {
	var data goesi.FacWarCorpJoinRequestMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.FactionID)
	return data, ids, nil
}

func (n facWarCorpJoinRequestMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s wants to join faction warfare", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has applied to join faction warfare on the side of %s.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type facWarCorpJoinWithdrawMsg struct {
	baseRenderer
}

func (n facWarCorpJoinWithdrawMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n facWarCorpJoinWithdrawMsg) unmarshal(text string) (goesi.FacWarCorpJoinWithdrawMsg, set.Set[int64], error) {
	var data goesi.FacWarCorpJoinWithdrawMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.FactionID)
	return data, ids, nil
}

func (n facWarCorpJoinWithdrawMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s withdrew faction warfare join application", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has withdrawn their application to join %s in faction warfare.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type facWarCorpLeaveRequestMsg struct {
	baseRenderer
}

func (n facWarCorpLeaveRequestMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n facWarCorpLeaveRequestMsg) unmarshal(text string) (goesi.FacWarCorpLeaveRequestMsg, set.Set[int64], error) {
	var data goesi.FacWarCorpLeaveRequestMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.FactionID)
	return data, ids, nil
}

func (n facWarCorpLeaveRequestMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s requested to leave faction warfare", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has applied to leave faction warfare from %s.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type facWarCorpLeaveWithdrawMsg struct {
	baseRenderer
}

func (n facWarCorpLeaveWithdrawMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n facWarCorpLeaveWithdrawMsg) unmarshal(text string) (goesi.FacWarCorpLeaveWithdrawMsg, set.Set[int64], error) {
	var data goesi.FacWarCorpLeaveWithdrawMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.FactionID)
	return data, ids, nil
}

func (n facWarCorpLeaveWithdrawMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s withdrew faction warfare leave application", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has withdrawn their application to leave %s in faction warfare.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type facWarLPDisqualifiedEvent struct {
	baseRenderer
}

func (n facWarLPDisqualifiedEvent) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n facWarLPDisqualifiedEvent) unmarshal(text string) (goesi.FacWarLPDisqualifiedEvent, set.Set[int64], error) {
	var data goesi.FacWarLPDisqualifiedEvent
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n facWarLPDisqualifiedEvent) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Faction warfare LP disqualified (event)"
	body = fmt.Sprintf(
		"**%d** loyalty points from a faction warfare event have been disqualified for %s.",
		data.Amount,
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type facWarLPDisqualifiedKill struct {
	baseRenderer
}

func (n facWarLPDisqualifiedKill) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n facWarLPDisqualifiedKill) unmarshal(text string) (goesi.FacWarLPDisqualifiedKill, set.Set[int64], error) {
	var data goesi.FacWarLPDisqualifiedKill
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n facWarLPDisqualifiedKill) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Faction warfare LP disqualified (kill)"
	body = fmt.Sprintf(
		"**%d** loyalty points from a faction warfare kill have been disqualified for %s.",
		data.Amount,
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type facWarLPPayoutEvent struct {
	baseRenderer
}

func (n facWarLPPayoutEvent) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n facWarLPPayoutEvent) unmarshal(text string) (goesi.FacWarLPPayoutEvent, set.Set[int64], error) {
	var data goesi.FacWarLPPayoutEvent
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n facWarLPPayoutEvent) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Faction warfare LP payout (event)"
	body = fmt.Sprintf(
		"**%d** loyalty points from a faction warfare event have been paid out to %s.",
		data.Amount,
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type facWarLPPayoutKill struct {
	baseRenderer
}

func (n facWarLPPayoutKill) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n facWarLPPayoutKill) unmarshal(text string) (goesi.FacWarLPPayoutKill, set.Set[int64], error) {
	var data goesi.FacWarLPPayoutKill
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n facWarLPPayoutKill) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Faction warfare LP payout (kill)"
	body = fmt.Sprintf(
		"**%d** loyalty points from a faction warfare kill have been paid out to %s.",
		data.Amount,
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}
