package service

import (
	"context"
	"errors"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"fmt"
)

func (s *Service) GetOrCreateEveCharacterESI(id int32) (model.EveCharacter, error) {
	ctx := context.Background()
	return s.getOrCreateEveCharacterESI(ctx, id)
}

func (s *Service) getOrCreateEveCharacterESI(ctx context.Context, id int32) (model.EveCharacter, error) {
	x, err := s.r.GetEveCharacter(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveCharacterFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveCharacterFromESI(ctx context.Context, id int32) (model.EveCharacter, error) {
	var dummy model.EveCharacter
	key := fmt.Sprintf("createEveCharacterFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, id, nil)
		if err != nil {
			return dummy, err
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
			return dummy, err
		}
		if err := s.updateRacesESI(ctx); err != nil {
			return dummy, err
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
			return dummy, err
		}
		return s.r.GetEveCharacter(ctx, id)
	})
	if err != nil {
		return dummy, err
	}
	return y.(model.EveCharacter), nil
}
