package evenotification

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/notification"
	"gopkg.in/yaml.v3"
)

func (s *EveNotificationService) renderMoonMining(ctx context.Context, type_, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case MoonminingExtractionStarted:
		title.Set("Moon mining extraction started")
		var data notification.MoonminingExtractionStarted
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		structureText, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("A moon mining extraction has been started %s.\n\n"+
			"The chunk will be ready on location at %s, "+
			"and will fracture automatically on %s.\n",
			structureText,
			fromLDAPTime(data.ReadyTime).Format(app.TimeDefaultFormat),
			fromLDAPTime(data.AutoTime).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case MoonminingExtractionFinished:
		title.Set("Moon mining extraction finished")
		var data notification.MoonminingExtractionFinished
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		structureText, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("The extraction %s "+
			"is finished and the chunk is ready to be shot at.\n\n"+
			"The chunk will automatically fracture on %s.",
			structureText,
			fromLDAPTime(data.AutoTime).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case MoonminingAutomaticFracture:
		title.Set("Moon mining automatic fracture")
		var data notification.MoonminingAutomaticFracture
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		structureText, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("The moon drill fitted to %s "+
			"has automatically fired and the moon products are ready to be harvested.",
			structureText,
		)
		body.Set(out)

	case MoonminingExtractionCancelled:
		title.Set("Moon mining extraction cancelled")
		var data notification.MoonminingExtractionCancelled
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		structureText, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		cancelledBy, err := s.EveUniverseService.GetOrCreateEveEntityESI(ctx, data.CancelledBy)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("An ongoing extraction for %s "+
			"has been cancelled by %s.",
			structureText,
			makeEveEntityProfileLink(cancelledBy),
		)
		body.Set(out)

	case MoonminingLaserFired:
		title.Set("Moon mining laser fired")
		var data notification.MoonminingLaserFired
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		structureText, err := s.makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName)
		if err != nil {
			return title, body, err
		}
		firedBy, err := s.EveUniverseService.GetOrCreateEveEntityESI(ctx, data.FiredBy)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("The moon drill fitted to %s"+
			"has been fired by %s "+
			"and the moon products are ready to be harvested.",
			structureText,
			makeEveEntityProfileLink(firedBy),
		)
		body.Set(out)
	}
	return title, body, nil
}

func (s *EveNotificationService) makeMoonMiningBaseText(ctx context.Context, moonID int32, structureName string) (string, error) {
	moon, err := s.EveUniverseService.GetOrCreateEveMoonESI(ctx, moonID)
	if err != nil {
		return "", err
	}
	out := fmt.Sprintf(
		"for **%s** at %s in %s",
		structureName,
		moon.Name,
		makeLocationLink(moon.SolarSystem),
	)
	return out, nil
}

// func (s* EveNotificationService) makeOreText(ctx context.Context, ores map[int32]float64) (string, err){
// 	entities, err := s.EveUniverseService.ToEveEntities(ctx, 0)
// }
