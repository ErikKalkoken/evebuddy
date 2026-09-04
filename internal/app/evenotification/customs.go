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

type containerPasswordMsg struct {
	baseRenderer
}

func (n containerPasswordMsg) unmarshal(text string) (goesi.ContainerPasswordMsg, set.Set[int64], error) {
	var data goesi.ContainerPasswordMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CharID, data.StationID), nil
}

func (n containerPasswordMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n containerPasswordMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	containerType, err := n.eus.GetOrCreateTypeESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Container password for %s", makeInfoLink(containerType))
	body := fmt.Sprintf(
		"%s has requested the %s password for a %s located at %s in %s.\n\n"+
			"Password: **%s**",
		makeInfoLink(entities[data.CharID]),
		data.PasswordType,
		makeInfoLink(containerType),
		makeInfoLink(entities[data.StationID]),
		makeInfoLink(solarSystem),
		data.Password,
	)
	return title, body, nil
}

type customsMsg struct {
	baseRenderer
}

func (n customsMsg) unmarshal(text string) (goesi.CustomsMsg, set.Set[int64], error) {
	var data goesi.CustomsMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.FactionID)
	for _, r := range data.LostList {
		ids.Add(r.TypeID)
	}
	return data, ids, nil
}

func (n customsMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n customsMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
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
	title := fmt.Sprintf("Customs office attack warning in %s", solarSystem.Name)
	var action string
	switch {
	case data.ShouldAttack != 0:
		action = "will be attacked"
	case data.ShouldConfiscate != 0:
		action = "will have its goods confiscated"
	default:
		action = "is at risk"
	}
	var lost []string
	var totalFine float64
	for _, r := range data.LostList {
		lost = append(lost, fmt.Sprintf("%dx %s", r.Quantity, entities[r.TypeID].Name))
		totalFine += r.Fine
	}
	body := fmt.Sprintf(
		"Your standing with %s has dropped below the required level for **%s** in %s.\n\n"+
			"Your customs office %s.",
		makeInfoLink(entities[data.FactionID]),
		humanize.Ftoa(data.StandingDivision),
		makeInfoLink(solarSystem),
		action,
	)
	if len(lost) > 0 {
		body += fmt.Sprintf(
			"\n\nGoods at risk of confiscation (total fine **%s** ISK):\n\n%s",
			humanize.Commaf(totalFine),
			strings.Join(lost, "\n\n"),
		)
	}
	return title, body, nil
}
