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

type bountyClaimMsg struct {
	baseRenderer
}

func (n bountyClaimMsg) unmarshal(text string) (goesi.BountyClaimMsg, set.Set[int64], error) {
	var data goesi.BountyClaimMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n bountyClaimMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n bountyClaimMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Bounty claimed by %s", entities[data.CharID].Name)
	body := fmt.Sprintf(
		"%s has claimed a bounty of **%s** ISK on you.",
		makeInfoLink(entities[data.CharID]),
		humanize.Commaf(data.Amount),
	)
	return title, body, nil
}

type bountyESSShared struct {
	baseRenderer
}

func (n bountyESSShared) unmarshal(text string) (goesi.BountyESSShared, set.Set[int64], error) {
	var data goesi.BountyESSShared
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n bountyESSShared) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n bountyESSShared) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "ESS bank shared"
	body := fmt.Sprintf(
		"%s has shared the contents of an Encounter Surveillance System.\n\n"+
			"Your share is **%s** ISK out of a total of **%s** ISK.",
		makeInfoLink(entities[data.CharID]),
		humanize.Commaf(data.MyIsk),
		humanize.Commaf(data.TotalIsk),
	)
	return title, body, nil
}

type bountyESSTaken struct {
	baseRenderer
}

func (n bountyESSTaken) unmarshal(text string) (goesi.BountyESSTaken, set.Set[int64], error) {
	var data goesi.BountyESSTaken
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID), nil
}

func (n bountyESSTaken) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n bountyESSTaken) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "ESS bank taken"
	body := fmt.Sprintf(
		"%s has taken **%s** ISK from an Encounter Surveillance System.\n\n"+
			"Your share of the loss is **%s** ISK.",
		makeInfoLink(entities[data.CharID]),
		humanize.Commaf(data.TotalIsk),
		humanize.Commaf(data.MyIsk),
	)
	return title, body, nil
}

type bountyPlacedAlliance struct {
	baseRenderer
}

func (n bountyPlacedAlliance) unmarshal(text string) (goesi.BountyPlacedAlliance, set.Set[int64], error) {
	var data goesi.BountyPlacedAlliance
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.BountyPlacerID), nil
}

func (n bountyPlacedAlliance) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n bountyPlacedAlliance) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Bounty of %s ISK placed on your alliance", humanize.Commaf(data.Bounty))
	body := fmt.Sprintf(
		"%s has placed a bounty of **%s** ISK on your alliance.",
		makeInfoLink(entities[data.BountyPlacerID]),
		humanize.Commaf(data.Bounty),
	)
	return title, body, nil
}

type bountyPlacedChar struct {
	baseRenderer
}

func (n bountyPlacedChar) unmarshal(text string) (goesi.BountyPlacedChar, set.Set[int64], error) {
	var data goesi.BountyPlacedChar
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.BountyPlacerID), nil
}

func (n bountyPlacedChar) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n bountyPlacedChar) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Bounty of %s ISK placed on you", humanize.Commaf(data.Bounty))
	body := fmt.Sprintf(
		"%s has placed a bounty of **%s** ISK on you.",
		makeInfoLink(entities[data.BountyPlacerID]),
		humanize.Commaf(data.Bounty),
	)
	return title, body, nil
}

type bountyPlacedCorp struct {
	baseRenderer
}

func (n bountyPlacedCorp) unmarshal(text string) (goesi.BountyPlacedCorp, set.Set[int64], error) {
	var data goesi.BountyPlacedCorp
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.BountyPlacerID), nil
}

func (n bountyPlacedCorp) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n bountyPlacedCorp) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Bounty of %s ISK placed on your corporation", humanize.Commaf(data.Bounty))
	body := fmt.Sprintf(
		"%s has placed a bounty of **%s** ISK on your corporation.",
		makeInfoLink(entities[data.BountyPlacerID]),
		humanize.Commaf(data.Bounty),
	)
	return title, body, nil
}

type bountyYourBountyClaimed struct {
	baseRenderer
}

func (n bountyYourBountyClaimed) unmarshal(text string) (goesi.BountyYourBountyClaimed, set.Set[int64], error) {
	var data goesi.BountyYourBountyClaimed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.VictimID), nil
}

func (n bountyYourBountyClaimed) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n bountyYourBountyClaimed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Bounty of %s ISK claimed", humanize.Commaf(data.Bounty))
	body := fmt.Sprintf(
		"Your bounty of **%s** ISK on %s has been claimed.",
		humanize.Commaf(data.Bounty),
		makeInfoLink(entities[data.VictimID]),
	)
	return title, body, nil
}
