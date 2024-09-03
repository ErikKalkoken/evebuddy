package evenotification

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/pkg/optional"
	"github.com/antihax/goesi/notification"
	"github.com/dustin/go-humanize"
	"gopkg.in/yaml.v3"
)

func (s *EveNotificationService) renderMoonMining(ctx context.Context, type_ Type, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case MoonminingExtractionStarted:
		var data notification.MoonminingExtractionStarted
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s has started extraction", data.StructureName))
		ores, err := s.makeOreText(ctx, data.OreVolumeByType)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("A moon mining extraction has been started %s.\n\n"+
			"The chunk will be ready on location at %s, "+
			"and will fracture automatically on %s.\n\n%s",
			o.text,
			fromLDAPTime(data.ReadyTime).Format(app.TimeDefaultFormat),
			fromLDAPTime(data.AutoTime).Format(app.TimeDefaultFormat),
			ores,
		)
		body.Set(out)

	case MoonminingExtractionFinished:
		var data notification.MoonminingExtractionFinished
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s has finished extraction", data.StructureName))
		ores, err := s.makeOreText(ctx, data.OreVolumeByType)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("The extraction %s "+
			"is finished and the chunk is ready to be shot at.\n\n"+
			"The chunk will automatically fracture on %s.\n\n%s",
			o.text,
			fromLDAPTime(data.AutoTime).Format(app.TimeDefaultFormat),
			ores,
		)
		body.Set(out)

	case MoonminingAutomaticFracture:
		var data notification.MoonminingAutomaticFracture
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s has auto fractured", data.StructureName))
		ores, err := s.makeOreText(ctx, data.OreVolumeByType)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("The moon drill fitted to %s "+
			"has automatically fired and the moon products are ready to be harvested.\n\n%s",
			o.text,
			ores,
		)
		body.Set(out)

	case MoonminingExtractionCancelled:
		var data notification.MoonminingExtractionCancelled
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s has canceled it's extraction", data.StructureName))
		cancelledBy := ""
		if data.CancelledBy != 0 {
			x, err := s.EveUniverseService.GetOrCreateEveEntityESI(ctx, data.CancelledBy)
			if err != nil {
				return title, body, err
			}
			cancelledBy = fmt.Sprintf(" by %s", makeEveEntityProfileLink(x))
		}
		out := fmt.Sprintf(
			"An ongoing extraction for %s has been cancelled%s.",
			o.text,
			cancelledBy,
		)
		body.Set(out)

	case MoonminingLaserFired:
		var data notification.MoonminingLaserFired
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		o, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf("%s has fired it's moon drill", data.StructureName))
		firedBy := ""
		if data.FiredBy != 0 {
			x, err := s.EveUniverseService.GetOrCreateEveEntityESI(ctx, data.FiredBy)
			if err != nil {
				return title, body, err
			}
			firedBy = fmt.Sprintf("by %s ", makeEveEntityProfileLink(x))
		}

		ores, err := s.makeOreText(ctx, data.OreVolumeByType)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf(
			"The moon drill fitted to %s has been fired %s"+
				"and the moon products are ready to be harvested.\n\n%s",
			o.text,
			firedBy,
			ores,
		)
		body.Set(out)
	}
	return title, body, nil
}

type moonMiningInfo struct {
	moon *app.EveMoon
	text string
}

func (s *EveNotificationService) makeMoonMiningBaseText(ctx context.Context, moonID int32, structureName string) (moonMiningInfo, error) {
	moon, err := s.EveUniverseService.GetOrCreateEveMoonESI(ctx, moonID)
	if err != nil {
		return moonMiningInfo{}, err
	}
	text := fmt.Sprintf(
		"for **%s** at %s in %s",
		structureName,
		moon.Name,
		makeLocationLink(moon.SolarSystem),
	)
	x := moonMiningInfo{
		moon: moon,
		text: text,
	}
	return x, nil
}

type oreItem struct {
	id     int32
	name   string
	volume float64
}

func (s *EveNotificationService) makeOreText(ctx context.Context, ores map[int32]float64) (string, error) {
	ids := slices.Collect(maps.Keys(ores))
	entities, err := s.EveUniverseService.ToEveEntities(ctx, ids)
	if err != nil {
		return "", err
	}
	items := make([]oreItem, 0)
	for id, v := range ores {
		i := oreItem{
			id:     id,
			name:   entities[id].Name,
			volume: v,
		}
		items = append(items, i)
	}
	slices.SortFunc(items, func(a, b oreItem) int {
		return cmp.Compare(a.name, b.name)
	})
	lines := make([]string, 0)
	for i := range slices.Values(items) {
		text := fmt.Sprintf("%s: %s m3", i.name, humanize.Comma(int64(i.volume)))
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n\n"), nil
}
