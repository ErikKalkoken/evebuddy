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
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
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
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
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
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.InvokingCharID]),
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
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
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
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.CharID]),
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
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
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
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
	)
	body = out
	return title, body, nil
}

type buddyConnectContactAdd struct {
	baseRenderer
}

func (n buddyConnectContactAdd) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.BuddyConnectContactAdd
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Contact added"
	body := fmt.Sprintf(
		"You have been added as a contact with a standing of **%d**.\n\n%s",
		data.Level,
		data.Message,
	)
	return title, body, nil
}

type charMedalMsg struct {
	baseRenderer
}

func (n charMedalMsg) unmarshal(text string) (goesi.CharMedalMsg, set.Set[int64], error) {
	var data goesi.CharMedalMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID), nil
}

func (n charMedalMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n charMedalMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Medal awarded"
	body := fmt.Sprintf(
		"You have been awarded a medal by %s.\n\n> %s",
		makeInfoLink(entities[data.CorpID]),
		data.Reason,
	)
	return title, body, nil
}

type charTerminationMsg struct {
	baseRenderer
}

func (n charTerminationMsg) unmarshal(text string) (goesi.CharTerminationMsg, set.Set[int64], error) {
	var data goesi.CharTerminationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.CorpID), nil
}

func (n charTerminationMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n charTerminationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has been terminated", entities[data.CharID].Name)
	body := fmt.Sprintf(
		"%s's role of **%s** at %s has been terminated.",
		makeInfoLink(entities[data.CharID]),
		data.RoleName,
		makeInfoLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type corpAppAcceptMsg struct {
	baseRenderer
}

func (n corpAppAcceptMsg) unmarshal(text string) (goesi.CorpAppAcceptMsg, set.Set[int64], error) {
	var data goesi.CorpAppAcceptMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.CorpID), nil
}

func (n corpAppAcceptMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpAppAcceptMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Application to %s accepted", entities[data.CorpID].Name)
	body := fmt.Sprintf(
		"%s's application to join %s has been accepted:\n\n> %s",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
		data.ApplicationText,
	)
	return title, body, nil
}

type corpAppRejectMsg struct {
	baseRenderer
}

func (n corpAppRejectMsg) unmarshal(text string) (goesi.CorpAppRejectMsg, set.Set[int64], error) {
	var data goesi.CorpAppRejectMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.CorpID), nil
}

func (n corpAppRejectMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpAppRejectMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Application to %s rejected", entities[data.CorpID].Name)
	body := fmt.Sprintf(
		"%s's application to join %s has been rejected:\n\n> %s",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
		data.ApplicationText,
	)
	return title, body, nil
}

type corpKicked struct {
	baseRenderer
}

func (n corpKicked) unmarshal(text string) (goesi.CorpKicked, set.Set[int64], error) {
	var data goesi.CorpKicked
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID), nil
}

func (n corpKicked) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpKicked) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Kicked from %s", entities[data.CorpID].Name)
	body := fmt.Sprintf("You have been kicked from %s.", makeInfoLink(entities[data.CorpID]))
	return title, body, nil
}

type corpNewCEOMsg struct {
	baseRenderer
}

func (n corpNewCEOMsg) unmarshal(text string) (goesi.CorpNewCEOMsg, set.Set[int64], error) {
	var data goesi.CorpNewCEOMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.NewCeoID, data.OldCeoID), nil
}

func (n corpNewCEOMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpNewCEOMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has a new CEO", entities[data.CorpID].Name)
	body := fmt.Sprintf(
		"%s has appointed %s as its new CEO, replacing %s.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(entities[data.NewCeoID]),
		makeInfoLink(entities[data.OldCeoID]),
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
	return data.Subject, data.Body, nil
}

type corpDividendMsg struct {
	baseRenderer
}

func (n corpDividendMsg) unmarshal(text string) (goesi.CorpDividendMsg, set.Set[int64], error) {
	var data goesi.CorpDividendMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID), nil
}

func (n corpDividendMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpDividendMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Dividend paid by %s", entities[data.CorpID].Name)
	body := fmt.Sprintf(
		"%s has paid out a dividend of **%s** ISK.",
		makeInfoLink(entities[data.CorpID]),
		humanize.Commaf(data.Payout),
	)
	return title, body, nil
}

type corpLiquidationMsg struct {
	baseRenderer
}

func (n corpLiquidationMsg) unmarshal(text string) (goesi.CorpLiquidationMsg, set.Set[int64], error) {
	var data goesi.CorpLiquidationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID), nil
}

func (n corpLiquidationMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpLiquidationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s has been liquidated", entities[data.CorpID].Name)
	body := fmt.Sprintf(
		"%s has been liquidated, paying out **%s** ISK.",
		makeInfoLink(entities[data.CorpID]),
		humanize.Commaf(data.Payout),
	)
	return title, body, nil
}

type corpNewsMsg struct {
	baseRenderer
}

func (n corpNewsMsg) unmarshal(text string) (goesi.CorpNewsMsg, set.Set[int64], error) {
	var data goesi.CorpNewsMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID), nil
}

func (n corpNewsMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpNewsMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("News from %s", entities[data.CorpID].Name)
	body := fmt.Sprintf("%s\n\n%s", makeInfoLink(entities[data.CorpID]), data.Body)
	return title, body, nil
}

type corpTaxChangeMsg struct {
	baseRenderer
}

func (n corpTaxChangeMsg) unmarshal(text string) (goesi.CorpTaxChangeMsg, set.Set[int64], error) {
	var data goesi.CorpTaxChangeMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID), nil
}

func (n corpTaxChangeMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpTaxChangeMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s tax rate changed", entities[data.CorpID].Name)
	body := fmt.Sprintf(
		"%s has changed its tax rate from **%.1f%%** to **%.1f%%**.",
		makeInfoLink(entities[data.CorpID]),
		data.OldTaxRate*100,
		data.NewTaxRate*100,
	)
	return title, body, nil
}

type corpFriendlyFireDisableTimerCompleted struct {
	baseRenderer
}

func (n corpFriendlyFireDisableTimerCompleted) unmarshal(text string) (goesi.CorpFriendlyFireDisableTimerCompleted, set.Set[int64], error) {
	var data goesi.CorpFriendlyFireDisableTimerCompleted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID), nil
}

func (n corpFriendlyFireDisableTimerCompleted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpFriendlyFireDisableTimerCompleted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Friendly fire disabled"
	body := fmt.Sprintf("Friendly fire has been disabled for %s.", makeInfoLink(entities[data.CorpID]))
	return title, body, nil
}

type corpFriendlyFireDisableTimerStarted struct {
	baseRenderer
}

func (n corpFriendlyFireDisableTimerStarted) unmarshal(text string) (goesi.CorpFriendlyFireDisableTimerStarted, set.Set[int64], error) {
	var data goesi.CorpFriendlyFireDisableTimerStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.CorpID), nil
}

func (n corpFriendlyFireDisableTimerStarted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpFriendlyFireDisableTimerStarted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Friendly fire disable timer started"
	body := fmt.Sprintf(
		"%s has started the timer to disable friendly fire for %s. "+
			"It will complete at **%s**.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
		fromLDAPTime(data.TimeFinished).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type corpFriendlyFireEnableTimerCompleted struct {
	baseRenderer
}

func (n corpFriendlyFireEnableTimerCompleted) unmarshal(text string) (goesi.CorpFriendlyFireEnableTimerCompleted, set.Set[int64], error) {
	var data goesi.CorpFriendlyFireEnableTimerCompleted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID), nil
}

func (n corpFriendlyFireEnableTimerCompleted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpFriendlyFireEnableTimerCompleted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Friendly fire enabled"
	body := fmt.Sprintf("Friendly fire has been enabled for %s.", makeInfoLink(entities[data.CorpID]))
	return title, body, nil
}

type corpFriendlyFireEnableTimerStarted struct {
	baseRenderer
}

func (n corpFriendlyFireEnableTimerStarted) unmarshal(text string) (goesi.CorpFriendlyFireEnableTimerStarted, set.Set[int64], error) {
	var data goesi.CorpFriendlyFireEnableTimerStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.CorpID), nil
}

func (n corpFriendlyFireEnableTimerStarted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpFriendlyFireEnableTimerStarted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Friendly fire enable timer started"
	body := fmt.Sprintf(
		"%s has started the timer to enable friendly fire for %s. "+
			"It will complete at **%s**.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.CorpID]),
		fromLDAPTime(data.TimeFinished).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type giftReceived struct {
	baseRenderer
}

func (n giftReceived) unmarshal(text string) (goesi.GiftReceived, set.Set[int64], error) {
	var data goesi.GiftReceived
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.SenderCharID), nil
}

func (n giftReceived) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n giftReceived) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Gift received from %s", entities[data.SenderCharID].Name)
	body := fmt.Sprintf(
		"%s has sent you a gift (**%d**x).\n\n> %s",
		makeInfoLink(entities[data.SenderCharID]),
		data.Quantity,
		data.Message,
	)
	return title, body, nil
}

type corpBecameWarEligible struct {
	baseRenderer
}

func (n corpBecameWarEligible) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Corporation is now war eligible"
	body := "Your corporation has become eligible for war declarations."
	return title, body, nil
}

type corpNoLongerWarEligible struct {
	baseRenderer
}

func (n corpNoLongerWarEligible) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Corporation is no longer war eligible"
	body := "Your corporation is no longer eligible for war declarations."
	return title, body, nil
}

type corporationGoalCreated struct {
	baseRenderer
}

func (n corporationGoalCreated) unmarshal(text string) (notification2.CorporationGoalCreated, set.Set[int64], error) {
	var data notification2.CorporationGoalCreated
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CreatorID), nil
}

func (n corporationGoalCreated) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corporationGoalCreated) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("New corporation goal: %s", data.GoalName)
	body := fmt.Sprintf(
		"%s has created a new corporation goal: **%s**.",
		makeInfoLink(entities[data.CreatorID]),
		data.GoalName,
	)
	return title, body, nil
}

type corporationGoalCompleted struct {
	baseRenderer
}

func (n corporationGoalCompleted) unmarshal(text string) (notification2.CorporationGoalCompleted, set.Set[int64], error) {
	var data notification2.CorporationGoalCompleted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CreatorID), nil
}

func (n corporationGoalCompleted) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corporationGoalCompleted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Corporation goal completed: %s", data.GoalName)
	body := fmt.Sprintf(
		"The corporation goal **%s** created by %s has been completed.",
		data.GoalName,
		makeInfoLink(entities[data.CreatorID]),
	)
	return title, body, nil
}

type corporationGoalClosed struct {
	baseRenderer
}

func (n corporationGoalClosed) unmarshal(text string) (notification2.CorporationGoalClosed, set.Set[int64], error) {
	var data notification2.CorporationGoalClosed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CloserID, data.CreatorID), nil
}

func (n corporationGoalClosed) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corporationGoalClosed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Corporation goal closed: %s", data.GoalName)
	body := fmt.Sprintf(
		"%s has closed the corporation goal **%s** created by %s.",
		makeInfoLink(entities[data.CloserID]),
		data.GoalName,
		makeInfoLink(entities[data.CreatorID]),
	)
	return title, body, nil
}

type corpVoteCEORevokedMsg struct {
	baseRenderer
}

func (n corpVoteCEORevokedMsg) unmarshal(text string) (notification2.CorpVoteCEORevokedMsg, set.Set[int64], error) {
	var data notification2.CorpVoteCEORevokedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n corpVoteCEORevokedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n corpVoteCEORevokedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "CEO revoked by corporation vote"
	body := fmt.Sprintf(
		"%s has been removed as CEO by a corporation vote.",
		makeInfoLink(entities[data.CharID]),
	)
	return title, body, nil
}
