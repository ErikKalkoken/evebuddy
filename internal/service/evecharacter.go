package service

import (
	"context"
	"errors"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"
)

func (s *Service) GetOrCreateEveCharacterESI(id int32) (*model.EveCharacter, error) {
	ctx := context.Background()
	return s.getOrCreateEveCharacterESI(ctx, id)
}

func (s *Service) getOrCreateEveCharacterESI(ctx context.Context, id int32) (*model.EveCharacter, error) {
	x, err := s.r.GetEveCharacter(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveCharacterFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveCharacterFromESI(ctx context.Context, id int32) (*model.EveCharacter, error) {
	key := fmt.Sprintf("createEveCharacterFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		ids := []int32{id, r.CorporationId}
		if r.AllianceId != 0 {
			ids = append(ids, r.AllianceId)
		}
		if r.FactionId != 0 {
			ids = append(ids, r.FactionId)
		}
		_, err = s.AddMissingEveEntities(ctx, ids)
		if err != nil {
			return nil, err
		}
		if err := s.updateRacesESI(ctx); err != nil {
			return nil, err
		}
		arg := storage.CreateEveCharacterParams{
			AllianceID:     r.AllianceId,
			ID:             id,
			Birthday:       r.Birthday,
			CorporationID:  r.CorporationId,
			Description:    r.Description,
			FactionID:      r.FactionId,
			Gender:         r.Gender,
			Name:           r.Name,
			RaceID:         r.RaceId,
			SecurityStatus: float64(r.SecurityStatus),
		}
		if err := s.r.CreateEveCharacter(ctx, arg); err != nil {
			return nil, err
		}
		return s.r.GetEveCharacter(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveCharacter), nil
}

// UpdateEveCharactersESI updates all known Eve characters from ESI.
func (s *Service) UpdateEveCharactersESI() error {
	ctx := context.Background()
	ids, err := s.r.ListEveCharacterIDs(ctx)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	slog.Info("Started updating eve characters", "count", len(ids))
	g := new(errgroup.Group)
	g.SetLimit(10)
	for _, id := range ids {
		id := id
		g.Go(func() error {
			return s.updateEveCharacterESI(ctx, id)
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to update EveCharacters: %w", err)
	}
	slog.Info("Finished updating eve characters", "count", len(ids))
	return nil
}

func (s *Service) updateEveCharacterESI(ctx context.Context, characterID int32) error {
	c, err := s.r.GetEveCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		rr, _, err := s.esiClient.ESI.CharacterApi.PostCharactersAffiliation(ctx, []int32{c.ID}, nil)
		if err != nil {
			return err
		}
		if len(rr) == 0 {
			return nil
		}
		r := rr[0]
		entityIDs := []int32{c.ID}
		entityIDs = append(entityIDs, r.CorporationId)
		if r.AllianceId != 0 {
			entityIDs = append(entityIDs, r.AllianceId)
		}
		if r.FactionId != 0 {
			entityIDs = append(entityIDs, r.FactionId)
		}
		_, err = s.AddMissingEveEntities(ctx, entityIDs)
		if err != nil {
			return err
		}
		corporation, err := s.r.GetEveEntity(ctx, r.CorporationId)
		if err != nil {
			return err
		}
		c.Corporation = corporation
		if r.AllianceId != 0 {
			alliance, err := s.r.GetEveEntity(ctx, r.AllianceId)
			if err != nil {
				return err
			}
			c.Alliance = alliance
		}
		if r.FactionId != 0 {
			faction, err := s.r.GetEveEntity(ctx, r.FactionId)
			if err != nil {
				return err
			}
			c.Faction = faction
		}
		return nil
	})
	g.Go(func() error {
		r2, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		c.SecurityStatus = float64(r2.SecurityStatus)
		return nil
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to update EveCharacter %d: %w", c.ID, err)
	}
	if err := s.r.UpdateEveCharacter(ctx, c); err != nil {
		return err
	}
	slog.Info("Updated eve character from ESI", "characterID", c.ID)
	return nil
}
