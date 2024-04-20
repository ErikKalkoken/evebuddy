package service

import (
	"context"
	"example/evebuddy/internal/helper/set"
	"log/slog"
)

func (s *Service) updateRacesESI(ctx context.Context) error {
	races, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRaces(ctx, nil)
	if err != nil {
		return err
	}
	ids, err := s.r.ListRaceIDs(ctx)
	if err != nil {
		return err
	}
	currentIDs := set.NewFromSlice(ids)
	count := 0
	for _, r := range races {
		if currentIDs.Has(r.RaceId) {
			continue
		}
		_, err := s.r.CreateRace(ctx, r.RaceId, r.Description, r.Name)
		if err != nil {
			return err
		}
		count++
	}
	if count > 0 {
		slog.Info("Created races", "count", count)
	}
	return nil
}
