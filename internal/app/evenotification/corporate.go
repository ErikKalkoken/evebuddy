package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

type charAppAcceptMsg struct {
	baseRenderer
}

func (n charAppAcceptMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n charAppAcceptMsg) unmarshal(text string) (notification.CharAppAcceptMsg, setInt32, error) {
	var data notification.CharAppAcceptMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n charAppAcceptMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n corpAppNewMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n corpAppNewMsg) unmarshal(text string) (notification.CorpAppNewMsg, setInt32, error) {
	var data notification.CorpAppNewMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n corpAppNewMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n corpAppInvitedMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n corpAppInvitedMsg) unmarshal(text string) (notification.CorpAppInvitedMsg, setInt32, error) {
	var data notification.CorpAppInvitedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID, data.CorpID, data.InvokingCharID)
	return data, ids, nil
}

func (n corpAppInvitedMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n charAppRejectMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n charAppRejectMsg) unmarshal(text string) (notification.CharAppRejectMsg, setInt32, error) {
	var data notification.CharAppRejectMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n charAppRejectMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n corpAppRejectCustomMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n corpAppRejectCustomMsg) unmarshal(text string) (notification.CorpAppRejectCustomMsg, setInt32, error) {
	var data notification.CorpAppRejectCustomMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n corpAppRejectCustomMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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
		"%s has rejected application from %s:\n\n>%s",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.CharID]),
		data.ApplicationText,
	)
	if data.CustomMessage != "" {
		out += fmt.Sprintf("\n\nReply:\n\n>%s", data.CustomMessage)
	}
	body = out
	return title, body, nil
}

type charAppWithdrawMsg struct {
	baseRenderer
}

func (n charAppWithdrawMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n charAppWithdrawMsg) unmarshal(text string) (notification.CharAppWithdrawMsg, setInt32, error) {
	var data notification.CharAppWithdrawMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n charAppWithdrawMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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

func (n charLeftCorpMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n charLeftCorpMsg) unmarshal(text string) (notification.CharLeftCorpMsg, setInt32, error) {
	var data notification.CharLeftCorpMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n charLeftCorpMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
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
