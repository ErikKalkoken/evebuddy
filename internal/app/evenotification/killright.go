package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"
)

type killReportFinalBlow struct {
	baseRenderer
}

func (n killReportFinalBlow) unmarshal(text string) (goesi.KillReportFinalBlow, set.Set[int64], error) {
	var data goesi.KillReportFinalBlow
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.VictimID), nil
}

func (n killReportFinalBlow) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n killReportFinalBlow) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	victimShip, err := n.eus.GetOrCreateTypeESI(ctx, data.VictimShipTypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Final blow on %s", entities[data.VictimID].Name)
	body := fmt.Sprintf(
		"You landed the final blow on %s, destroying their %s.",
		makeInfoLink(entities[data.VictimID]),
		makeInfoLink(victimShip),
	)
	return title, body, nil
}

type killReportVictim struct {
	baseRenderer
}

func (n killReportVictim) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.KillReportVictim
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	victimShip, err := n.eus.GetOrCreateTypeESI(ctx, data.VictimShipTypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Your %s was destroyed", victimShip.Name)
	body := fmt.Sprintf("Your %s was destroyed.", makeInfoLink(victimShip))
	return title, body, nil
}

type killRightAvailable struct {
	baseRenderer
}

func (n killRightAvailable) unmarshal(text string) (goesi.KillRightAvailable, set.Set[int64], error) {
	var data goesi.KillRightAvailable
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.ToEntityID), nil
}

func (n killRightAvailable) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n killRightAvailable) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Kill right available for %s", entities[data.CharID].Name)
	body := fmt.Sprintf(
		"A kill right against %s is now available to %s for **%s** ISK.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.ToEntityID]),
		humanize.Commaf(data.Price),
	)
	return title, body, nil
}

type killRightAvailableOpen struct {
	baseRenderer
}

func (n killRightAvailableOpen) unmarshal(text string) (goesi.KillRightAvailableOpen, set.Set[int64], error) {
	var data goesi.KillRightAvailableOpen
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n killRightAvailableOpen) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n killRightAvailableOpen) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Kill right available for %s", entities[data.CharID].Name)
	body := fmt.Sprintf(
		"A kill right against %s is now available to anyone for **%s** ISK.",
		makeInfoLink(entities[data.CharID]),
		humanize.Commaf(data.Price),
	)
	return title, body, nil
}

type killRightEarned struct {
	baseRenderer
}

func (n killRightEarned) unmarshal(text string) (goesi.KillRightEarned, set.Set[int64], error) {
	var data goesi.KillRightEarned
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n killRightEarned) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n killRightEarned) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Kill right earned against %s", entities[data.CharID].Name)
	body := fmt.Sprintf(
		"You have earned a kill right against %s.",
		makeInfoLink(entities[data.CharID]),
	)
	return title, body, nil
}

type killRightUnavailable struct {
	baseRenderer
}

func (n killRightUnavailable) unmarshal(text string) (goesi.KillRightUnavailable, set.Set[int64], error) {
	var data goesi.KillRightUnavailable
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.ToEntityID), nil
}

func (n killRightUnavailable) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n killRightUnavailable) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Kill right against %s no longer available", entities[data.CharID].Name)
	body := fmt.Sprintf(
		"The kill right against %s that was available to %s is no longer available.",
		makeInfoLink(entities[data.CharID]),
		makeInfoLink(entities[data.ToEntityID]),
	)
	return title, body, nil
}

type killRightUnavailableOpen struct {
	baseRenderer
}

func (n killRightUnavailableOpen) unmarshal(text string) (goesi.KillRightUnavailableOpen, set.Set[int64], error) {
	var data goesi.KillRightUnavailableOpen
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n killRightUnavailableOpen) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n killRightUnavailableOpen) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Kill right against %s no longer available", entities[data.CharID].Name)
	body := fmt.Sprintf(
		"The kill right against %s that was available to anyone is no longer available.",
		makeInfoLink(entities[data.CharID]),
	)
	return title, body, nil
}

type killRightUsed struct {
	baseRenderer
}

func (n killRightUsed) unmarshal(text string) (goesi.KillRightUsed, set.Set[int64], error) {
	var data goesi.KillRightUsed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n killRightUsed) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n killRightUsed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Kill right against %s used", entities[data.CharID].Name)
	body := fmt.Sprintf(
		"A kill right against %s has been used.",
		makeInfoLink(entities[data.CharID]),
	)
	return title, body, nil
}
