package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
)

type charMedalMsg struct {
	baseRenderer
}

func (n charMedalMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n charMedalMsg) unmarshal(text string) (goesi.CharMedalMsg, set.Set[int64], error) {
	var data goesi.CharMedalMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID)
	return data, ids, nil
}

func (n charMedalMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Medal awarded by %s", entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"You have been awarded a medal by %s.\n\n> %s",
		makeEveEntityProfileLink(entities[data.CorpID]),
		data.Reason,
	)
	return title, body, nil
}

type charTerminationMsg struct {
	baseRenderer
}

func (n charTerminationMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n charTerminationMsg) unmarshal(text string) (goesi.CharTerminationMsg, set.Set[int64], error) {
	var data goesi.CharTerminationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.CorpID)
	return data, ids, nil
}

func (n charTerminationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has been removed from %s", entities[data.CharID].Name, entities[data.CorpID].Name)
	body = fmt.Sprintf(
		"%s has had their role **%s** revoked and been removed from %s.",
		makeEveEntityProfileLink(entities[data.CharID]),
		data.RoleName,
		makeEveEntityProfileLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type cloneActivationMsg struct {
	baseRenderer
}

func (n cloneActivationMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n cloneActivationMsg) unmarshal(text string) (goesi.CloneActivationMsg, set.Set[int64], error) {
	var data goesi.CloneActivationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.PodKillerID)
	return data, ids, nil
}

func (n cloneActivationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Your clone has been activated"
	body = fmt.Sprintf(
		"Your clone has been activated after your pod was destroyed by %s. "+
			"You have lost **%d** skill points.",
		makeEveEntityProfileLink(entities[data.PodKillerID]),
		data.SkillPointsLost,
	)
	return title, body, nil
}

type cloneActivationMsg2 struct {
	baseRenderer
}

func (n cloneActivationMsg2) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n cloneActivationMsg2) unmarshal(text string) (goesi.CloneActivationMsg2, set.Set[int64], error) {
	var data goesi.CloneActivationMsg2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.PodKillerID)
	return data, ids, nil
}

func (n cloneActivationMsg2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Your clone has been activated"
	body = fmt.Sprintf(
		"Your clone has been activated after your pod was destroyed by %s.",
		makeEveEntityProfileLink(entities[data.PodKillerID]),
	)
	return title, body, nil
}

type cloneMovedMsg struct {
	baseRenderer
}

func (n cloneMovedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n cloneMovedMsg) unmarshal(text string) (goesi.CloneMovedMsg, set.Set[int64], error) {
	var data goesi.CloneMovedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CorpID, data.StationID, data.NewStationID)
	return data, ids, nil
}

func (n cloneMovedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Clone moved"
	body = fmt.Sprintf(
		"A clone belonging to %s has been moved from **%s** to **%s**.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		entities[data.StationID].Name,
		entities[data.NewStationID].Name,
	)
	return title, body, nil
}

type cloneRevokedMsg1 struct {
	baseRenderer
}

func (n cloneRevokedMsg1) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.CloneRevokedMsg1
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	station, err := n.eus.GetOrCreateEntityESI(ctx, data.StationID)
	if err != nil {
		return "", "", err
	}
	newStation, err := n.eus.GetOrCreateEntityESI(ctx, data.NewStationID)
	if err != nil {
		return "", "", err
	}
	title := "Clone revoked"
	body := fmt.Sprintf(
		"A clone at **%s** has been revoked and moved to **%s**.",
		station.Name,
		newStation.Name,
	)
	return title, body, nil
}

type cloneRevokedMsg2 struct {
	baseRenderer
}

func (n cloneRevokedMsg2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.CloneRevokedMsg2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	station, err := n.eus.GetOrCreateEntityESI(ctx, data.StationID)
	if err != nil {
		return "", "", err
	}
	newStation, err := n.eus.GetOrCreateEntityESI(ctx, data.NewStationID)
	if err != nil {
		return "", "", err
	}
	title := "Clone revoked"
	body := fmt.Sprintf(
		"A clone at **%s** has been revoked and moved to **%s**.",
		station.Name,
		newStation.Name,
	)
	return title, body, nil
}

type jumpCloneDeletedMsg1 struct {
	baseRenderer
}

func (n jumpCloneDeletedMsg1) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.JumpCloneDeletedMsg1
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	location, err := n.eus.GetOrCreateEntityESI(ctx, data.LocationID)
	if err != nil {
		return "", "", err
	}
	title := "Jump clone deleted"
	body := fmt.Sprintf(
		"One of your jump clones at **%s** has been deleted.",
		location.Name,
	)
	return title, body, nil
}

type jumpCloneDeletedMsg2 struct {
	baseRenderer
}

func (n jumpCloneDeletedMsg2) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n jumpCloneDeletedMsg2) unmarshal(text string) (goesi.JumpCloneDeletedMsg2, set.Set[int64], error) {
	var data goesi.JumpCloneDeletedMsg2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.DestroyerID, data.LocationOwnerID)
	return data, ids, nil
}

func (n jumpCloneDeletedMsg2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	location, err := n.eus.GetOrCreateEntityESI(ctx, data.LocationID)
	if err != nil {
		return title, body, err
	}
	title = "Jump clone deleted"
	body = fmt.Sprintf(
		"One of your jump clones at **%s** has been deleted by %s.",
		location.Name,
		makeEveEntityProfileLink(entities[data.DestroyerID]),
	)
	return title, body, nil
}

type agentRetiredTrigravian struct {
	baseRenderer
}

func (n agentRetiredTrigravian) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.AgentRetiredTrigravian
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	agent, err := n.eus.GetOrCreateEntityESI(ctx, data.AgentID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Triglavian agent %s retired", agent.Name)
	body := fmt.Sprintf(
		"The Triglavian agent **%s** has been retired and is no longer available.",
		agent.Name,
	)
	return title, body, nil
}

type allMaintenanceBillMsg struct {
	baseRenderer
}

func (n allMaintenanceBillMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allMaintenanceBillMsg) unmarshal(text string) (goesi.AllMaintenanceBillMsg, set.Set[int64], error) {
	var data goesi.AllMaintenanceBillMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllianceID)
	return data, ids, nil
}

func (n allMaintenanceBillMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Alliance maintenance bill due for %s", entities[data.AllianceID].Name)
	body = fmt.Sprintf(
		"An alliance maintenance bill for %s is due on **%s**.",
		makeEveEntityProfileLink(entities[data.AllianceID]),
		fromLDAPTime(data.DueDate).Format(app.DateTimeFormat),
	)
	return title, body, nil
}
