package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"
)

type charAppAcceptMsg struct {
	baseRenderer
}

func (n charAppAcceptMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n charAppAcceptMsg) unmarshal(text string) (goesi.CharAppAcceptMsg, set.Set[int64], error) {
	var data goesi.CharAppAcceptMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n charAppAcceptMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
		"%s joins %s",
		entities[data.CharID].Name,
		entities[data.CorpID].Name,
	)
	out := fmt.Sprintf(
		"%s is now a member of %s.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	body = out
	return title, body, nil
}

type corpAppNewMsg struct {
	baseRenderer
}

func (n corpAppNewMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpAppNewMsg) unmarshal(text string) (goesi.CorpAppNewMsg, set.Set[int64], error) {
	var data goesi.CorpAppNewMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n corpAppNewMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("New application from %s", entities[data.CharID].Name)
	out := fmt.Sprintf(
		"New application from %s to join %s:\n\n> %s",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.CorpID]),
		data.ApplicationText,
	)
	body = out
	return title, body, nil
}

type corpAppInvitedMsg struct {
	baseRenderer
}

func (n corpAppInvitedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpAppInvitedMsg) unmarshal(text string) (goesi.CorpAppInvitedMsg, set.Set[int64], error) {
	var data goesi.CorpAppInvitedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID, data.InvokingCharID)
	return data, ids, nil
}

func (n corpAppInvitedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has been invited", entities[data.CharID].Name)
	out := fmt.Sprintf(
		"%s has been invited to join %s by %s:\n\n> %s",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.InvokingCharID]),
		data.ApplicationText,
	)
	body = out
	return title, body, nil
}

type charAppRejectMsg struct {
	baseRenderer
}

func (n charAppRejectMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n charAppRejectMsg) unmarshal(text string) (goesi.CharAppRejectMsg, set.Set[int64], error) {
	var data goesi.CharAppRejectMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n charAppRejectMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s rejected invitation", entities[data.CharID].Name)
	out := fmt.Sprintf(
		"Application from %s to join %s has been rejected:\n\n> %s",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.CorpID]),
		data.ApplicationText,
	)
	body = out
	return title, body, nil
}

type corpAppRejectCustomMsg struct {
	baseRenderer
}

func (n corpAppRejectCustomMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpAppRejectCustomMsg) unmarshal(text string) (goesi.CorpAppRejectCustomMsg, set.Set[int64], error) {
	var data goesi.CorpAppRejectCustomMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n corpAppRejectCustomMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Application from %s rejected", entities[data.CharID].Name)
	out := fmt.Sprintf(
		"%s has rejected application from %s:\n\n> %s",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.CharID]),
		data.ApplicationText,
	)
	if data.CustomMessage != "" {
		out += fmt.Sprintf("\n\nReply:\n\n> %s", data.CustomMessage)
	}
	body = out
	return title, body, nil
}

type charAppWithdrawMsg struct {
	baseRenderer
}

func (n charAppWithdrawMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n charAppWithdrawMsg) unmarshal(text string) (goesi.CharAppWithdrawMsg, set.Set[int64], error) {
	var data goesi.CharAppWithdrawMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n charAppWithdrawMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s withdrew application", entities[data.CharID].Name)
	out := fmt.Sprintf(
		"%s has withdrawn application to join %s:\n\n>%s",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.CharID]),
		data.ApplicationText,
	)
	body = out
	return title, body, nil
}

type charLeftCorpMsg struct {
	baseRenderer
}

func (n charLeftCorpMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n charLeftCorpMsg) unmarshal(text string) (goesi.CharLeftCorpMsg, set.Set[int64], error) {
	var data goesi.CharLeftCorpMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n charLeftCorpMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s left %s",
		entities[data.CharID].Name,
		entities[data.CorpID].Name,
	)
	out := fmt.Sprintf(
		"%s is no longer a member of %s.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	body = out
	return title, body, nil
}

type corpAppAcceptMsg struct {
	baseRenderer
}

func (n corpAppAcceptMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpAppAcceptMsg) unmarshal(text string) (goesi.CorpAppAcceptMsg, set.Set[int64], error) {
	var data goesi.CorpAppAcceptMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n corpAppAcceptMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has accepted and joined %s", entities[data.CharID].Name, entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has accepted their application and is now a member of %s.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type corpAppRejectMsg struct {
	baseRenderer
}

func (n corpAppRejectMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpAppRejectMsg) unmarshal(text string) (goesi.CorpAppRejectMsg, set.Set[int64], error) {
	var data goesi.CorpAppRejectMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n corpAppRejectMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Application from %s rejected", entities[data.CharID].Name)
	body = fmt.Sprintf(
		"%s has rejected the application from %s.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}

type corpDividendMsg struct {
	baseRenderer
}

func (n corpDividendMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpDividendMsg) unmarshal(text string) (goesi.CorpDividendMsg, set.Set[int64], error) {
	var data goesi.CorpDividendMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n corpDividendMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	corp := makeEveEntityProfileLink(entities[data.CorpID])
	recipientType := "shareholders"
	if data.IsMembers {
		recipientType = "members"
	}
	title = fmt.Sprintf("Dividend from %s", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has issued a dividend to %s. "+
			"Total amount paid: **%s** ISK (**%s** ISK per share).",
		corp,
		recipientType,
		humanize.Commaf(data.Amount),
		humanize.Commaf(data.Payout),
	)
	return title, body, nil
}

type corpFriendlyFireDisableTimerCompleted struct {
	baseRenderer
}

func (n corpFriendlyFireDisableTimerCompleted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpFriendlyFireDisableTimerCompleted) unmarshal(text string) (goesi.CorpFriendlyFireDisableTimerCompleted, set.Set[int64], error) {
	var data goesi.CorpFriendlyFireDisableTimerCompleted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n corpFriendlyFireDisableTimerCompleted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Friendly fire disabled"
	body = fmt.Sprintf(
		"The friendly fire disable timer for %s has completed. Friendly fire is now disabled.",
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type corpFriendlyFireDisableTimerStarted struct {
	baseRenderer
}

func (n corpFriendlyFireDisableTimerStarted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpFriendlyFireDisableTimerStarted) unmarshal(text string) (goesi.CorpFriendlyFireDisableTimerStarted, set.Set[int64], error) {
	var data goesi.CorpFriendlyFireDisableTimerStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n corpFriendlyFireDisableTimerStarted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Friendly fire disable timer started"
	body = fmt.Sprintf(
		"%s has started the friendly fire disable timer for %s. "+
			"Friendly fire will be disabled on **%s**.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.CorpID]),
		fromLDAPTime(data.TimeFinished).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type corpFriendlyFireEnableTimerCompleted struct {
	baseRenderer
}

func (n corpFriendlyFireEnableTimerCompleted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpFriendlyFireEnableTimerCompleted) unmarshal(text string) (goesi.CorpFriendlyFireEnableTimerCompleted, set.Set[int64], error) {
	var data goesi.CorpFriendlyFireEnableTimerCompleted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n corpFriendlyFireEnableTimerCompleted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Friendly fire enabled"
	body = fmt.Sprintf(
		"The friendly fire enable timer for %s has completed. Friendly fire is now enabled.",
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type corpFriendlyFireEnableTimerStarted struct {
	baseRenderer
}

func (n corpFriendlyFireEnableTimerStarted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpFriendlyFireEnableTimerStarted) unmarshal(text string) (goesi.CorpFriendlyFireEnableTimerStarted, set.Set[int64], error) {
	var data goesi.CorpFriendlyFireEnableTimerStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n corpFriendlyFireEnableTimerStarted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Friendly fire enable timer started"
	body = fmt.Sprintf(
		"%s has started the friendly fire enable timer for %s. "+
			"Friendly fire will be enabled on **%s**.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.CorpID]),
		fromLDAPTime(data.TimeFinished).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type corpKicked struct {
	baseRenderer
}

func (n corpKicked) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpKicked) unmarshal(text string) (goesi.CorpKicked, set.Set[int64], error) {
	var data goesi.CorpKicked
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n corpKicked) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has been kicked from its alliance", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has been forcibly removed from its alliance.",
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type corpLiquidationMsg struct {
	baseRenderer
}

func (n corpLiquidationMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpLiquidationMsg) unmarshal(text string) (goesi.CorpLiquidationMsg, set.Set[int64], error) {
	var data goesi.CorpLiquidationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n corpLiquidationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s is being liquidated", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s is going through liquidation. "+
			"Total amount: **%s** ISK, payout per share: **%s** ISK.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		humanize.Commaf(data.Amount),
		humanize.Commaf(data.Payout),
	)
	return title, body, nil
}

type corpNewCEOMsg struct {
	baseRenderer
}

func (n corpNewCEOMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpNewCEOMsg) unmarshal(text string) (goesi.CorpNewCEOMsg, set.Set[int64], error) {
	var data goesi.CorpNewCEOMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.NewCeoID, data.OldCeoID)
	return data, ids, nil
}

func (n corpNewCEOMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("New CEO for %s", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has become the new CEO of %s, replacing %s.",
		makeEveEntityProfileLink(entities[data.NewCeoID]),
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.OldCeoID]),
	)
	return title, body, nil
}

type corpNewsMsg struct {
	baseRenderer
}

func (n corpNewsMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpNewsMsg) unmarshal(text string) (goesi.CorpNewsMsg, set.Set[int64], error) {
	var data goesi.CorpNewsMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n corpNewsMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Corporation news from %s", entities[data.CorpID].Name)
	body = data.Body
	if body == "" {
		body = fmt.Sprintf("A news update has been posted by %s.", makeEveEntityProfileLink(entities[data.CorpID]))
	}
	return title, body, nil
}

type corpTaxChangeMsg struct {
	baseRenderer
}

func (n corpTaxChangeMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpTaxChangeMsg) unmarshal(text string) (goesi.CorpTaxChangeMsg, set.Set[int64], error) {
	var data goesi.CorpTaxChangeMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n corpTaxChangeMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Corporation tax changed for %s", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"The corporation tax rate for %s has been changed from **%.0f%%** to **%.0f%%**.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		data.OldTaxRate*100,
		data.NewTaxRate*100,
	)
	return title, body, nil
}

type corpVoteMsg struct {
	baseRenderer
}

func (n corpVoteMsg) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.CorpVoteMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Corporation vote: %s", data.Subject)
	body := data.Body
	if body == "" {
		body = fmt.Sprintf("A new corporation vote has been opened: **%s**.", data.Subject)
	}
	return title, body, nil
}

type corpVoteCEORevokedMsg struct {
	baseRenderer
}

func (n corpVoteCEORevokedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpVoteCEORevokedMsg) unmarshal(text string) (notification2.CorpVoteCEORevokedMsg, set.Set[int64], error) {
	var data notification2.CorpVoteCEORevokedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n corpVoteCEORevokedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "CEO election vote revoked"
	body = fmt.Sprintf(
		"The CEO election vote in %s started by %s has been revoked.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}

type corpStructLostMsg struct {
	baseRenderer
}

func (n corpStructLostMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.CorpStructLostMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarsystemID)
	if err != nil {
		return "", "", err
	}
	structureType, err := n.eus.GetOrCreateEntityESI(ctx, data.StructureTypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Corp structure lost in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"A **%s** structure belonging to your corporation has been lost in %s.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type corpOfficeExpirationMsg struct {
	baseRenderer
}

func (n corpOfficeExpirationMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpOfficeExpirationMsg) unmarshal(text string) (notification2.CorpOfficeExpirationMsg, set.Set[int64], error) {
	var data notification2.CorpOfficeExpirationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n corpOfficeExpirationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Office lease expiring for %s", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"The office lease for %s at **%s** will expire in **%d** days.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		data.StationName,
		data.DaysToExpiry,
	)
	return title, body, nil
}

type corporationGoalMsg struct {
	baseRenderer
}

func (n corporationGoalMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corporationGoalMsg) unmarshal(text string) (notification2.CorporationGoalMsg, set.Set[int64], error) {
	var data notification2.CorporationGoalMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorporationID, data.CreatorID)
	return data, ids, nil
}

type corporationGoalCreated struct {
	corporationGoalMsg
}

func (n corporationGoalCreated) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Corporation goal created: %s", data.GoalName)
	body = fmt.Sprintf(
		"A new corporation goal **%s** has been created by %s in %s.",
		data.GoalName,
		makeEveEntityProfileLink(entities[data.CreatorID]),
		makeEveEntityProfileLink(entities[data.CorporationID]),
	)
	return title, body, nil
}

type corporationGoalCompleted struct {
	corporationGoalMsg
}

func (n corporationGoalCompleted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Corporation goal completed: %s", data.GoalName)
	body = fmt.Sprintf(
		"The corporation goal **%s** in %s has been completed.",
		data.GoalName,
		makeEveEntityProfileLink(entities[data.CorporationID]),
	)
	return title, body, nil
}

type corporationGoalClosed struct {
	corporationGoalMsg
}

func (n corporationGoalClosed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Corporation goal closed: %s", data.GoalName)
	body = fmt.Sprintf(
		"The corporation goal **%s** in %s has been closed.",
		data.GoalName,
		makeEveEntityProfileLink(entities[data.CorporationID]),
	)
	return title, body, nil
}

type corporationGoalNameChange struct {
	baseRenderer
}

func (n corporationGoalNameChange) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corporationGoalNameChange) unmarshal(text string) (notification2.CorporationGoalNameChange, set.Set[int64], error) {
	var data notification2.CorporationGoalNameChange
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorporationID)
	return data, ids, nil
}

func (n corporationGoalNameChange) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Corporation goal renamed: %s", data.NewName)
	body = fmt.Sprintf(
		"A corporation goal in %s has been renamed from **%s** to **%s**.",
		makeEveEntityProfileLink(entities[data.CorporationID]),
		data.OldName,
		data.NewName,
	)
	return title, body, nil
}

type corporationLeft struct {
	baseRenderer
}

func (n corporationLeft) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corporationLeft) unmarshal(text string) (notification2.CorporationLeft, set.Set[int64], error) {
	var data notification2.CorporationLeft
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllianceID, data.CorpID)
	return data, ids, nil
}

func (n corporationLeft) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has left %s", entities[data.CorpID].Name, entities[data.AllianceID].Name)
	body = fmt.Sprintf(
		"%s has left the alliance %s.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.AllianceID]),
	)
	return title, body, nil
}

type officeLeaseCanceledInsufficientStandings struct {
	baseRenderer
}

func (n officeLeaseCanceledInsufficientStandings) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n officeLeaseCanceledInsufficientStandings) unmarshal(text string) (notification2.OfficeLeaseCanceledInsufficientStandings, set.Set[int64], error) {
	var data notification2.OfficeLeaseCanceledInsufficientStandings
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n officeLeaseCanceledInsufficientStandings) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Office lease cancelled due to insufficient standings"
	body = fmt.Sprintf(
		"The office lease for %s has been cancelled due to insufficient standings with the station owner.",
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}
