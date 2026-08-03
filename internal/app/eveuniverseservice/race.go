package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
)

func (s *EVEUniverseService) GetOrCreateRaceESI(ctx context.Context, raceID int64) (*app.EveRace, error) {
	o, err, _ := xsingleflight.Do(&s.sfg, fmt.Sprintf("GetOrCreateRaceESI-%d", raceID), func() (*app.EveRace, error) {
		o, err := s.st.GetEveRace(ctx, raceID)
		if err == nil {
			return o, nil
		}
		if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		races, _, err := s.esiClient.UniverseAPI.GetUniverseRaces(ctx).Execute()
		if err != nil {
			return nil, err
		}
		for _, r := range races {
			if r.RaceId != raceID {
				continue
			}
			_, err := s.AddMissingEntities(ctx, set.Of(r.AllianceId))
			if err != nil {
				return nil, err
			}
			if err := s.st.CreateEveRace(ctx, storage.CreateEveRaceParams{
				ID:          r.RaceId,
				Description: r.Description,
				Name:        r.Name,
				FactionID:   optional.New(r.AllianceId),
			}); err != nil {
				return nil, err
			}
			slog.Info("Created eve race", "id", raceID)
			o, err := s.st.GetEveRace(ctx, raceID)
			if err != nil {
				return nil, err
			}
			return o, nil
		}
		return nil, fmt.Errorf("race ID %d: %w", raceID, app.ErrNotFound)
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *EVEUniverseService) GetOrCreateBloodlineESI(ctx context.Context, id int64) (*app.EveBloodline, error) {
	o, err, _ := xsingleflight.Do(&s.sfg, fmt.Sprintf("GetOrCreateBloodlineESI-%d", id), func() (*app.EveBloodline, error) {
		o, err := s.st.GetEveBloodline(ctx, id)
		if err == nil {
			return o, nil
		}
		if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		bloodlines, _, err := s.esiClient.UniverseAPI.GetUniverseBloodlines(ctx).Execute()
		if err != nil {
			return nil, err
		}
		for _, bl := range bloodlines {
			if bl.BloodlineId != id {
				continue
			}
			corporation, err := s.GetOrCreateEntityESI(ctx, bl.CorporationId)
			if err != nil {
				return nil, err
			}
			race, err := s.GetOrCreateRaceESI(ctx, bl.RaceId)
			if err != nil {
				return nil, err
			}
			arg := storage.CreateEveBloodlineParams{
				Charisma:      optional.FromZeroValue(bl.Charisma),
				CorporationID: corporation.ID,
				Description:   bl.Description,
				ID:            bl.BloodlineId,
				Intelligence:  optional.FromZeroValue(bl.Intelligence),
				Memory:        optional.FromZeroValue(bl.Memory),
				Name:          bl.Name,
				Perception:    optional.FromZeroValue(bl.Perception),
				RaceID:        race.ID,
				Willpower:     optional.FromZeroValue(bl.Willpower),
			}
			if bl.ShipTypeId > 0 {
				et, err := s.GetOrCreateTypeESI(ctx, bl.ShipTypeId)
				if err != nil {
					return nil, err
				}
				arg.ShipTypeID.Set(et.ID)
			}
			if err := s.st.CreateEveBloodline(ctx, arg); err != nil {
				return nil, err
			}
			slog.Info("Created eve bloodline", "id", id)
			o, err := s.st.GetEveBloodline(ctx, bl.BloodlineId)
			if err != nil {
				return nil, err
			}
			return o, nil
		}
		return nil, fmt.Errorf("bloodline ID %d: %w", id, app.ErrNotFound)
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}
