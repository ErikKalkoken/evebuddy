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
		out := fmt.Sprintf("A moon mining extraction has been started %s."+
			"The chunk will be ready on location at %s, "+
			"and will fracture automatically on %s.\n",
			structureText,
			fromLDAPTime(data.ReadyTime).Format(app.TimeDefaultFormat),
			fromLDAPTime(data.AutoTime).Format(app.TimeDefaultFormat),
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
