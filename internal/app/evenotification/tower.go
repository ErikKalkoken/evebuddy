package evenotification

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"github.com/dustin/go-humanize"
	"gopkg.in/yaml.v3"
)

func (s *EveNotificationService) renderTower(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	switch type_ {
	case TowerAlertMsg:
		return s.renderTowerAlertMsg(ctx, text)
	case TowerResourceAlertMsg:
		return s.renderTowerResourceAlertMsg(ctx, text)
	}
	return optional.Optional[string]{}, optional.Optional[string]{}, fmt.Errorf("render tower: unknown notification type: %s", type_)
}

func (s *EveNotificationService) renderTowerAlertMsg(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.TowerAlertMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := s.makeTowerBaseText(ctx, data.MoonID, data.TypeID)
	if err != nil {
		return title, body, err
	}
	entities, err := s.eus.ToEntities(ctx, []int32{data.AggressorAllianceID, data.AggressorCorpID, data.AggressorID})
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("Starbase at %s is under attack", o.moon.Name))
	b := fmt.Sprintf(
		"%s is under attack.\n\n"+
			"Attacking Character: %s\n\n"+
			"Attacking Corporation: %s",
		o.intro,
		makeEveEntityProfileLink(entities[data.AggressorID]),
		makeEveEntityProfileLink(entities[data.AggressorCorpID]),
	)
	if data.AggressorAllianceID != 0 {
		b += fmt.Sprintf(
			"\n\nAttacking Alliance: %s",
			makeEveEntityProfileLink(entities[data.AggressorAllianceID]),
		)
	}
	body.Set(b)
	return title, body, nil
}

func (s *EveNotificationService) renderTowerResourceAlertMsg(ctx context.Context, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	var data notification.TowerResourceAlertMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := s.makeTowerBaseText(ctx, data.MoonID, data.TypeID)
	if err != nil {
		return title, body, err
	}
	title.Set(fmt.Sprintf("Starbase at %s is running out of fuel", o.moon.Name))
	b := fmt.Sprintf("%s is running out of fuel in less then 24hrs.\n\n", o.intro)
	if len(data.Wants) > 0 {
		b += fmt.Sprintf(
			"Fuel remaining: %s units", humanize.Comma(int64(data.Wants[0].Quantity)),
		)
	}
	body.Set(b)
	return title, body, nil
}

type towerInfo struct {
	type_ *app.EveType
	moon  *app.EveMoon
	intro string
}

func (s *EveNotificationService) makeTowerBaseText(ctx context.Context, moonID, typeID int32) (towerInfo, error) {
	structureType, err := s.eus.GetOrCreateTypeESI(ctx, typeID)
	if err != nil {
		return towerInfo{}, err
	}
	moon, err := s.eus.GetOrCreateMoonESI(ctx, moonID)
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
