package evenotification

import (
	"context"
	"fmt"
	"time"

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
	title := fmt.Sprintf("Insurance expiring for %s", data.ShipName)
	body := fmt.Sprintf(
		"The insurance policy for your ship **%s** expires on **%s**.",
		data.ShipName,
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
	shipType, err := n.eus.GetOrCreateEntityESI(ctx, data.ShipTypeID)
	if err != nil {
		return "", "", err
	}
	title := "First ship insurance issued"
	var body string
	if data.IsHouseWarmingGift != 0 {
		body = fmt.Sprintf(
			"Congratulations on your new **%s**! It has been insured as a welcome gift from CCP.",
			shipType.Name,
		)
	} else {
		body = fmt.Sprintf(
			"Your first **%s** has been insured.",
			shipType.Name,
		)
	}
	return title, body, nil
}

type insuranceInvalidatedMsg struct {
	baseRenderer
}

func (n insuranceInvalidatedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.InsuranceInvalidatedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	shipType, err := n.eus.GetOrCreateEntityESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Insurance invalidated for %s", shipType.Name)
	body := fmt.Sprintf(
		"The insurance policy for your **%s** has been invalidated. "+
			"Policy was active from **%s** to **%s**.",
		shipType.Name,
		fromLDAPTime(data.StartDate).Format(app.DateTimeFormat),
		fromLDAPTime(data.EndDate).Format(app.DateTimeFormat),
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
	shipType, err := n.eus.GetOrCreateEntityESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Insurance issued for %s", data.ShipName)
	body := fmt.Sprintf(
		"An insurance policy has been issued for your **%s** (%s tier) for **%d** weeks.\n\n"+
			"Policy is valid from **%s** to **%s**.",
		data.ShipName,
		shipType.Name,
		data.NumWeeks,
		fromLDAPTime(data.StartDate).Format(app.DateTimeFormat),
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
	title := "Insurance payout received"
	body := fmt.Sprintf(
		"Your insurance policy has paid out **%s** ISK for the loss of your ship.",
		humanize.Commaf(data.Amount),
	)
	return title, body, nil
}
