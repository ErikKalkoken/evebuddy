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

func (n bountyClaimMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n bountyClaimMsg) unmarshal(text string) (goesi.BountyClaimMsg, set.Set[int64], error) {
	var data goesi.BountyClaimMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n bountyClaimMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Bounty payout received"
	body = fmt.Sprintf(
		"You have received a bounty payout of **%s** ISK for killing %s.",
		humanize.Commaf(data.Amount),
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}

type bountyESSShared struct {
	baseRenderer
}

func (n bountyESSShared) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n bountyESSShared) unmarshal(text string) (goesi.BountyESSShared, set.Set[int64], error) {
	var data goesi.BountyESSShared
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n bountyESSShared) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "ESS bounty shared"
	body = fmt.Sprintf(
		"You received **%s** ISK as your share from the ESS bounty payout (total: **%s** ISK), "+
			"triggered by %s.",
		humanize.Commaf(data.MyIsk),
		humanize.Commaf(data.TotalIsk),
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}

type bountyESSTaken struct {
	baseRenderer
}

func (n bountyESSTaken) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n bountyESSTaken) unmarshal(text string) (goesi.BountyESSTaken, set.Set[int64], error) {
	var data goesi.BountyESSTaken
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n bountyESSTaken) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "ESS bounty taken"
	body = fmt.Sprintf(
		"%s has robbed the ESS, taking **%s** ISK of a total **%s** ISK.",
		makeEveEntityProfileLink(entities[data.CharID]),
		humanize.Commaf(data.MyIsk),
		humanize.Commaf(data.TotalIsk),
	)
	return title, body, nil
}

type bountyPlacedAlliance struct {
	baseRenderer
}

func (n bountyPlacedAlliance) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n bountyPlacedAlliance) unmarshal(text string) (goesi.BountyPlacedAlliance, set.Set[int64], error) {
	var data goesi.BountyPlacedAlliance
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.BountyPlacerID)
	return data, ids, nil
}

func (n bountyPlacedAlliance) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Bounty placed on your alliance"
	body = fmt.Sprintf(
		"%s has placed a bounty of **%s** ISK on your alliance.",
		makeEveEntityProfileLink(entities[data.BountyPlacerID]),
		humanize.Commaf(data.Bounty),
	)
	return title, body, nil
}

type bountyPlacedChar struct {
	baseRenderer
}

func (n bountyPlacedChar) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n bountyPlacedChar) unmarshal(text string) (goesi.BountyPlacedChar, set.Set[int64], error) {
	var data goesi.BountyPlacedChar
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.BountyPlacerID)
	return data, ids, nil
}

func (n bountyPlacedChar) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Bounty placed on you"
	body = fmt.Sprintf(
		"%s has placed a bounty of **%s** ISK on you.",
		makeEveEntityProfileLink(entities[data.BountyPlacerID]),
		humanize.Commaf(data.Bounty),
	)
	return title, body, nil
}

type bountyPlacedCorp struct {
	baseRenderer
}

func (n bountyPlacedCorp) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n bountyPlacedCorp) unmarshal(text string) (goesi.BountyPlacedCorp, set.Set[int64], error) {
	var data goesi.BountyPlacedCorp
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.BountyPlacerID)
	return data, ids, nil
}

func (n bountyPlacedCorp) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Bounty placed on your corporation"
	body = fmt.Sprintf(
		"%s has placed a bounty of **%s** ISK on your corporation.",
		makeEveEntityProfileLink(entities[data.BountyPlacerID]),
		humanize.Commaf(data.Bounty),
	)
	return title, body, nil
}

type bountyYourBountyClaimed struct {
	baseRenderer
}

func (n bountyYourBountyClaimed) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n bountyYourBountyClaimed) unmarshal(text string) (goesi.BountyYourBountyClaimed, set.Set[int64], error) {
	var data goesi.BountyYourBountyClaimed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.VictimID)
	return data, ids, nil
}

func (n bountyYourBountyClaimed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Your bounty has been claimed"
	body = fmt.Sprintf(
		"Your bounty of **%s** ISK has been claimed by killing %s.",
		humanize.Commaf(data.Bounty),
		makeEveEntityProfileLink(entities[data.VictimID]),
	)
	return title, body, nil
}
