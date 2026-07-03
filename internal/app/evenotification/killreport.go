package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"
)

const zkillboardKillMailURL = "https://zkillboard.com/kill"

type killReportFinalBlow struct {
	baseRenderer
}

func (n killReportFinalBlow) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n killReportFinalBlow) unmarshal(text string) (goesi.KillReportFinalBlow, set.Set[int64], error) {
	var data goesi.KillReportFinalBlow
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.VictimID, data.VictimShipTypeID)
	return data, ids, nil
}

func (n killReportFinalBlow) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	victimName := entities[data.VictimID].Name
	shipTypeName := entities[data.VictimShipTypeID].Name
	title = fmt.Sprintf("You scored the final blow on %s", victimName)
	killURL := fmt.Sprintf("%s/%d/", zkillboardKillMailURL, data.KillMailID)
	body = fmt.Sprintf(
		"You scored the final blow on %s flying a **%s**. "+
			"[View kill report](%s)",
		makeEveEntityProfileLink(entities[data.VictimID]),
		shipTypeName,
		killURL,
	)
	return title, body, nil
}

type killReportVictim struct {
	baseRenderer
}

func (n killReportVictim) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n killReportVictim) unmarshal(text string) (goesi.KillReportVictim, set.Set[int64], error) {
	var data goesi.KillReportVictim
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.VictimShipTypeID)
	return data, ids, nil
}

func (n killReportVictim) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	shipTypeName := entities[data.VictimShipTypeID].Name
	title = fmt.Sprintf("Your %s has been destroyed", shipTypeName)
	killURL := fmt.Sprintf("%s/%d/", zkillboardKillMailURL, data.KillMailID)
	body = fmt.Sprintf(
		"Your **%s** has been destroyed. "+
			"[View kill report](%s)",
		shipTypeName,
		killURL,
	)
	return title, body, nil
}

type killRightAvailable struct {
	baseRenderer
}

func (n killRightAvailable) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n killRightAvailable) unmarshal(text string) (goesi.KillRightAvailable, set.Set[int64], error) {
	var data goesi.KillRightAvailable
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.ToEntityID)
	return data, ids, nil
}

func (n killRightAvailable) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Kill right available against %s", entities[data.CharID].Name)
	body = fmt.Sprintf(
		"A kill right against %s is now available to %s for **%.2f** ISK.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.ToEntityID]),
		data.Price,
	)
	return title, body, nil
}

type killRightAvailableOpen struct {
	baseRenderer
}

func (n killRightAvailableOpen) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n killRightAvailableOpen) unmarshal(text string) (goesi.KillRightAvailableOpen, set.Set[int64], error) {
	var data goesi.KillRightAvailableOpen
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n killRightAvailableOpen) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Kill right available against %s (open)", entities[data.CharID].Name)
	body = fmt.Sprintf(
		"A kill right against %s is now publicly available for **%.2f** ISK.",
		makeEveEntityProfileLink(entities[data.CharID]),
		data.Price,
	)
	return title, body, nil
}

type killRightEarned struct {
	baseRenderer
}

func (n killRightEarned) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n killRightEarned) unmarshal(text string) (goesi.KillRightEarned, set.Set[int64], error) {
	var data goesi.KillRightEarned
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n killRightEarned) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Kill right earned against %s", entities[data.CharID].Name)
	body = fmt.Sprintf(
		"You have earned a kill right against %s.",
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}

type killRightUnavailable struct {
	baseRenderer
}

func (n killRightUnavailable) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n killRightUnavailable) unmarshal(text string) (goesi.KillRightUnavailable, set.Set[int64], error) {
	var data goesi.KillRightUnavailable
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.ToEntityID)
	return data, ids, nil
}

func (n killRightUnavailable) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Kill right against %s is no longer available", entities[data.CharID].Name)
	body = fmt.Sprintf(
		"The kill right against %s that was available to %s is no longer available.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeEveEntityProfileLink(entities[data.ToEntityID]),
	)
	return title, body, nil
}

type killRightUnavailableOpen struct {
	baseRenderer
}

func (n killRightUnavailableOpen) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n killRightUnavailableOpen) unmarshal(text string) (goesi.KillRightUnavailableOpen, set.Set[int64], error) {
	var data goesi.KillRightUnavailableOpen
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n killRightUnavailableOpen) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Public kill right against %s expired", entities[data.CharID].Name)
	body = fmt.Sprintf(
		"The public kill right against %s is no longer available.",
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}

type killRightUsed struct {
	baseRenderer
}

func (n killRightUsed) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n killRightUsed) unmarshal(text string) (goesi.KillRightUsed, set.Set[int64], error) {
	var data goesi.KillRightUsed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n killRightUsed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Your kill right against %s has been used", entities[data.CharID].Name)
	body = fmt.Sprintf(
		"Your kill right against %s has been activated and used.",
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}
