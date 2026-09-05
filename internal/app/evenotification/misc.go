package evenotification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"
)

type incursionCompletedMsg struct {
	baseRenderer
}

func (n incursionCompletedMsg) unmarshal(text string) (goesi.IncursionCompletedMsg, set.Set[int64], error) {
	var data goesi.IncursionCompletedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Set[int64]{}
	for _, r := range data.TopTen {
		if len(r) > 0 {
			ids.Add(r[0])
		}
	}
	return data, ids, nil
}

func (n incursionCompletedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n incursionCompletedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Incursion in %s completed", solarSystem.Name)
	var top []string
	for i, r := range data.TopTen {
		if i >= 3 || len(r) < 2 {
			break
		}
		top = append(top, fmt.Sprintf("%s (**%s** ISK)", makeInfoLink(entities[r[0]]), humanize.Commaf(float64(r[1]))))
	}
	body := fmt.Sprintf("The incursion in %s has been completed.", makeInfoLink(solarSystem))
	if len(top) > 0 {
		body += "\n\nTop payouts:\n\n" + strings.Join(top, "\n\n")
	}
	return title, body, nil
}

type industryTeamAuctionLost struct {
	baseRenderer
}

func (n industryTeamAuctionLost) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.IndustryTeamAuctionLost
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := "Industry team auction lost"
	body := fmt.Sprintf(
		"You did not win the auction for an industry team requested for a job in %s.\n\n"+
			"The winning bid was **%s** ISK. Your bid was **%s** ISK.",
		makeInfoLink(solarSystem),
		humanize.Commaf(data.TotalIsk),
		humanize.Commaf(data.YourAmount),
	)
	return title, body, nil
}

type locateCharMsg struct {
	baseRenderer
}

func (n locateCharMsg) unmarshal(text string) (goesi.LocateCharMsg, set.Set[int64], error) {
	var data goesi.LocateCharMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharacterID), nil
}

func (n locateCharMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n locateCharMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.TargetLocation.SolarSystem)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s located", entities[data.CharacterID].Name)
	body := fmt.Sprintf(
		"%s has been located in %s.",
		makeInfoLink(entities[data.CharacterID]),
		makeInfoLink(solarSystem),
	)
	return title, body, nil
}

type missionOfferExpirationMsg struct {
	baseRenderer
}

func (n missionOfferExpirationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.MissionOfferExpirationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	rewardType, err := n.eus.GetOrCreateTypeESI(ctx, data.MissionKeywords.RewardTypeID)
	if err != nil {
		return "", "", err
	}
	title := "Mission offer expiring"
	body := fmt.Sprintf(
		"%s\n\nReward: **%d**x %s.",
		strings.Join(data.Body, "\n"),
		data.MissionKeywords.RewardQuantity,
		makeInfoLink(rewardType),
	)
	return title, body, nil
}

type oldLscMessages struct {
	baseRenderer
}

func (n oldLscMessages) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.OldLscMessages
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	return data.Subject, data.Body, nil
}

type operationFinished struct {
	baseRenderer
}

func (n operationFinished) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.OperationFinished
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Operation finished"
	body := fmt.Sprintf(
		"An operation you participated in has finished, rewarding you **%s** ISK.",
		humanize.Comma(int64(data.Rewards.Isk)),
	)
	return title, body, nil
}

type reimbursementMsg struct {
	baseRenderer
}

func (n reimbursementMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.ReimbursementMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	shipType, err := n.eus.GetOrCreateTypeESI(ctx, data.ShipTypeID)
	if err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Reimbursement for %s", makeInfoLink(shipType))
	body := fmt.Sprintf(
		"You have been reimbursed for the loss of your %s in %s.",
		makeInfoLink(shipType),
		makeInfoLink(solarSystem),
	)
	return title, body, nil
}

type researchMissionAvailableMsg struct {
	baseRenderer
}

func (n researchMissionAvailableMsg) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Research mission available"
	body := "A new research mission is available from one of your research agents."
	return title, body, nil
}

type seasonalChallengeCompleted struct {
	baseRenderer
}

func (n seasonalChallengeCompleted) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.SeasonalChallengeCompleted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Seasonal challenge completed"
	body := fmt.Sprintf(
		"You have completed a seasonal challenge and been awarded **%d** points.",
		data.PointsAwarded,
	)
	return title, body, nil
}
