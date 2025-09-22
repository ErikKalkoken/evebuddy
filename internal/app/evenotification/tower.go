package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/antihax/goesi/notification"
	"github.com/dustin/go-humanize"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type towerInfo struct {
	type_ *app.EveType
	moon  *app.EveMoon
	intro string
}

func makeTowerBaseText(ctx context.Context, moonID, typeID int32, eus EveUniverseService) (towerInfo, error) {
	structureType, err := eus.GetOrCreateTypeESI(ctx, typeID)
	if err != nil {
		return towerInfo{}, err
	}
	moon, err := eus.GetOrCreateMoonESI(ctx, moonID)
	if err != nil {
		return towerInfo{}, err
	}
	intro := fmt.Sprintf("The %s at %s in %s ", structureType.Name, moon.Name, makeSolarSystemLink(moon.SolarSystem))
	x := towerInfo{
		type_: structureType,
		moon:  moon,
		intro: intro,
	}
	return x, nil
}

type towerAlertMsg struct {
	baseRenderer
}

func (n towerAlertMsg) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n towerAlertMsg) unmarshal(text string) (notification.TowerAlertMsg, setInt32, error) {
	var data notification.TowerAlertMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Of(data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID)
	return data, ids, nil
}

func (n towerAlertMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	o, err := makeTowerBaseText(ctx, data.MoonID, data.TypeID, n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Starbase at %s is under attack", o.moon.Name)
	b := fmt.Sprintf(
		"%s is under attack.\n\n"+
			"Aggressing Pilot: %s\n\n"+
			"Aggressing Pilot's Corporation: %s",
		o.intro,
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.AggressorCorpID]),
	)
	if data.AggressorAllianceID != 0 {
		b += fmt.Sprintf(
			"\n\nAggressing Pilot's Alliance: %s",
			makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
		)
	}
	body = b
	return title, body, nil
}

type towerResourceAlertMsg struct {
	baseRenderer
}

func (n towerResourceAlertMsg) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.TowerResourceAlertMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeTowerBaseText(ctx, data.MoonID, data.TypeID, n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Starbase at %s is running out of fuel", o.moon.Name)
	b := fmt.Sprintf("%s is running out of fuel in less then 24hrs.\n\n", o.intro)
	if len(data.Wants) > 0 {
		b += fmt.Sprintf(
			"Fuel remaining: %s units", humanize.Comma(int64(data.Wants[0].Quantity)),
		)
	}
	body = b
	return title, body, nil
}
