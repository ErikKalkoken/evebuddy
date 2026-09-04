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
)

type insuranceExpirationMsg struct {
	baseRenderer
}

func (n insuranceExpirationMsg) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.InsuranceExpirationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Insurance for %s about to expire", data.ShipName)
	body := fmt.Sprintf(
		"The insurance contract for your ship **%s**, "+
			"taken out on %s, will expire on %s.",
		data.ShipName,
		fromLDAPTime(data.StartDate).Format(app.DateTimeFormat),
		fromLDAPTime(data.EndDate).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type insuranceFirstShipMsg struct {
	baseRenderer
}

func (n insuranceFirstShipMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.InsuranceFirstShipMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	shipType, err := n.eus.GetOrCreateTypeESI(ctx, data.ShipTypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Insurance issued for your first %s", shipType.Name)
	body := fmt.Sprintf("You have taken out insurance for your first %s.", makeInfoLink(shipType))
	if data.IsHouseWarmingGift != 0 {
		body += " This insurance was a house warming gift."
	}
	return title, body, nil
}

type insuranceInvalidatedMsg struct {
	baseRenderer
}

func (n insuranceInvalidatedMsg) unmarshal(text string) (goesi.InsuranceInvalidatedMsg, set.Set[int64], error) {
	var data goesi.InsuranceInvalidatedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.OwnerID), nil
}

func (n insuranceInvalidatedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n insuranceInvalidatedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	shipType, err := n.eus.GetOrCreateTypeESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Insurance for %s invalidated", makeInfoLink(shipType))
	body := fmt.Sprintf(
		"The insurance contract taken out by %s for a %s "+
			"on %s has been invalidated.",
		makeInfoLink(entities[data.OwnerID]),
		makeInfoLink(shipType),
		fromLDAPTime(data.StartDate).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type insuranceIssuedMsg struct {
	baseRenderer
}

func (n insuranceIssuedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.InsuranceIssuedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	shipType, err := n.eus.GetOrCreateTypeESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Insurance issued for %s", data.ShipName)
	body := fmt.Sprintf(
		"An insurance contract for your %s **%s** has been issued, "+
			"covering **%.0f%%** of its value for **%d** week(s), "+
			"expiring on %s.",
		makeInfoLink(shipType),
		data.ShipName,
		data.Level*100,
		data.NumWeeks,
		fromLDAPTime(data.EndDate).Format(app.DateTimeFormat),
	)
	return title, body, nil
}

type insurancePayoutMsg struct {
	baseRenderer
}

func (n insurancePayoutMsg) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.InsurancePayoutMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Insurance payout of %s ISK", humanize.Commaf(data.Amount))
	body := fmt.Sprintf(
		"An insurance payout of **%s** ISK for the loss of your ship has been made.",
		humanize.Commaf(data.Amount),
	)
	return title, body, nil
}
