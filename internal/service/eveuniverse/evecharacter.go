package eveuniverse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (eu *EveUniverse) GetOrCreateEveCharacterESI(ctx context.Context, id int32) (*model.EveCharacter, error) {
	x, err := eu.st.GetEveCharacter(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveCharacterFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverse) createEveCharacterFromESI(ctx context.Context, id int32) (*model.EveCharacter, error) {
	key := fmt.Sprintf("createEveCharacterFromESI-%d", id)
	y, err, _ := eu.sfg.Do(key, func() (any, error) {
		r, _, err := eu.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, id, nil)
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
		_, err = eu.AddMissingEveEntities(ctx, ids)
		if err != nil {
			return nil, err
		}
		_, err = eu.GetOrCreateEveRaceESI(ctx, r.RaceId)
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
		if err := eu.st.CreateEveCharacter(ctx, arg); err != nil {
			return nil, err
		}
		return eu.st.GetEveCharacter(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveCharacter), nil
}

// UpdateAllEveCharactersESI updates all known Eve characters from ESI.
func (eu *EveUniverse) UpdateAllEveCharactersESI(ctx context.Context) error {
	ids, err := eu.st.ListEveCharacterIDs(ctx)
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
			return eu.updateEveCharacterESI(ctx, id)
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to update EveCharacters: %w", err)
	}
	slog.Info("Finished updating eve characters", "count", len(ids))
	return nil
}

func (eu *EveUniverse) updateEveCharacterESI(ctx context.Context, characterID int32) error {
	c, err := eu.st.GetEveCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		rr, _, err := eu.esiClient.ESI.CharacterApi.PostCharactersAffiliation(ctx, []int32{c.ID}, nil)
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
		_, err = eu.AddMissingEveEntities(ctx, entityIDs)
		if err != nil {
			return err
		}
		corporation, err := eu.st.GetEveEntity(ctx, r.CorporationId)
		if err != nil {
			return err
		}
		c.Corporation = corporation
		if r.AllianceId != 0 {
			alliance, err := eu.st.GetEveEntity(ctx, r.AllianceId)
			if err != nil {
				return err
			}
			c.Alliance = alliance
		}
		if r.FactionId != 0 {
			faction, err := eu.st.GetEveEntity(ctx, r.FactionId)
			if err != nil {
				return err
			}
			c.Faction = faction
		}
		return nil
	})
	g.Go(func() error {
		r2, _, err := eu.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		c.Description = r2.Description
		c.SecurityStatus = float64(r2.SecurityStatus)
		c.Title = r2.Title
		return nil
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to update EveCharacter %d: %w", c.ID, err)
	}
	if err := eu.st.UpdateEveCharacter(ctx, c); err != nil {
		return err
	}
	slog.Info("Updated eve character from ESI", "characterID", c.ID)
	return nil
}
