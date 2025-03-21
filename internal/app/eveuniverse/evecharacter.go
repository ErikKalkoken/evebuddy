package eveuniverse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/antihax/goesi/esi"
)

func (s *EveUniverseService) GetOrCreateCharacterESI(ctx context.Context, id int32) (*app.EveCharacter, error) {
	x, err := s.st.GetEveCharacter(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.createEveCharacterFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *EveUniverseService) GetCharacterESI(ctx context.Context, characterID int32) (*app.EveCharacter, error) {
	c, err := s.fetchEveCharacterfromESI(ctx, characterID)
	if err != nil {
		return nil, err
	}
	_, err = s.AddMissingEntities(ctx, []int32{characterID, c.AllianceId, c.CorporationId, c.FactionId})
	if err != nil {
		return nil, err
	}
	o := &app.EveCharacter{
		Birthday:       c.Birthday,
		Description:    c.Description,
		Gender:         c.Gender,
		ID:             characterID,
		Name:           c.Name,
		SecurityStatus: float64(c.SecurityStatus),
		Title:          c.Title,
	}
	o.Corporation, err = s.getValidEveEntity(ctx, c.CorporationId)
	if err != nil {
		return nil, err
	}
	o.Race, err = s.st.GetEveRace(ctx, c.RaceId)
	if err != nil {
		return nil, err
	}
	o.Alliance, err = s.getValidEveEntity(ctx, c.AllianceId)
	if err != nil {
		return nil, err
	}
	o.Faction, err = s.getValidEveEntity(ctx, c.FactionId)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *EveUniverseService) createEveCharacterFromESI(ctx context.Context, id int32) (*app.EveCharacter, error) {
	key := fmt.Sprintf("createEveCharacterFromESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		r, err := s.fetchEveCharacterfromESI(ctx, id)
		if err != nil {
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
			Title:          r.Title,
		}
		if err := s.st.CreateEveCharacter(ctx, arg); err != nil {
			return nil, err
		}
		return s.st.GetEveCharacter(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveCharacter), nil
}

func (s *EveUniverseService) fetchEveCharacterfromESI(ctx context.Context, id int32) (esi.GetCharactersCharacterIdOk, error) {
	r, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, id, nil)
	if err != nil {
		return esi.GetCharactersCharacterIdOk{}, err
	}
	ids := []int32{id, r.CorporationId}
	if r.AllianceId != 0 {
		ids = append(ids, r.AllianceId)
	}
	if r.FactionId != 0 {
		ids = append(ids, r.FactionId)
	}
	_, err = s.AddMissingEntities(ctx, ids)
	if err != nil {
		return esi.GetCharactersCharacterIdOk{}, err
	}
	_, err = s.GetOrCreateEveRaceESI(ctx, r.RaceId)
	if err != nil {
		return esi.GetCharactersCharacterIdOk{}, err
	}
	return r, nil
}

// UpdateAllEveCharactersESI updates all known Eve characters from ESI.
func (s *EveUniverseService) UpdateAllEveCharactersESI(ctx context.Context) error {
	ids, err := s.st.ListEveCharacterIDs(ctx)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	slog.Info("Started updating eve characters", "count", len(ids))
	g := new(errgroup.Group)
	g.SetLimit(10)
	for id := range ids.Values() {
		id := id
		g.Go(func() error {
			return s.updateEveCharacterESI(ctx, id)
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("update EveCharacters: %w", err)
	}
	slog.Info("Finished updating eve characters", "count", len(ids))
	return nil
}

func (s *EveUniverseService) updateEveCharacterESI(ctx context.Context, characterID int32) error {
	c, err := s.st.GetEveCharacter(ctx, characterID)
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
		_, err = s.AddMissingEntities(ctx, entityIDs)
		if err != nil {
			return err
		}
		corporation, err := s.st.GetEveEntity(ctx, r.CorporationId)
		if err != nil {
			return err
		}
		c.Corporation = corporation
		if r.AllianceId != 0 {
			alliance, err := s.st.GetEveEntity(ctx, r.AllianceId)
			if err != nil {
				return err
			}
			c.Alliance = alliance
		}
		if r.FactionId != 0 {
			faction, err := s.st.GetEveEntity(ctx, r.FactionId)
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
		c.Description = r2.Description
		c.SecurityStatus = float64(r2.SecurityStatus)
		c.Title = r2.Title
		return nil
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("update EveCharacter %d: %w", c.ID, err)
	}
	if err := s.st.UpdateEveCharacter(ctx, c); err != nil {
		return err
	}
	slog.Info("Updated eve character from ESI", "characterID", c.ID)
	return nil
}
