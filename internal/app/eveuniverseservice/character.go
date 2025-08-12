package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/icrowley/fake"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func (s *EveUniverseService) GetCharacterESI(ctx context.Context, characterID int32) (*app.EveCharacter, error) {
	return s.st.GetEveCharacter(ctx, characterID)
}

func (s *EveUniverseService) GetOrCreateCharacterESI(ctx context.Context, characterID int32) (*app.EveCharacter, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateCharacterESI-%d", characterID), func() (any, error) {
		o, err := s.st.GetEveCharacter(ctx, characterID)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		r, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, characterID, nil)
		if err != nil {
			return nil, err
		}
		ids := set.Of(characterID, r.CorporationId)
		if r.AllianceId != 0 {
			ids.Add(r.AllianceId)
		}
		if r.FactionId != 0 {
			ids.Add(r.FactionId)
		}
		_, err = s.AddMissingEntities(ctx, ids)
		if err != nil {
			return nil, err
		}
		_, err = s.GetOrCreateRaceESI(ctx, r.RaceId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveCharacterParams{
			AllianceID:     r.AllianceId,
			ID:             characterID,
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
		slog.Info("Created eve character", "ID", characterID)
		return s.st.GetEveCharacter(ctx, characterID)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveCharacter), nil
}

// RandomizeAllCharacterNames randomizes the names of all characters.
func (s *EveUniverseService) RandomizeAllCharacterNames(ctx context.Context) error {
	ids, err := s.st.ListEveCharacterIDs(ctx)
	if err != nil {
		return err
	}
	if ids.Size() == 0 {
		return nil
	}
	for id := range ids.All() {
		name := fake.FullName()
		err := s.st.UpdateEveCharacterName(ctx, id, name)
		if err != nil {
			return err
		}
		err = s.updateEntityNameIfExists(ctx, id, name)
		if err != nil {
			return err
		}
	}
	return s.scs.UpdateCharacters(ctx)
}

// UpdateAllCharactersESI updates all known Eve characters from ESI
// and returns the IDs of all changed characters.
func (s *EveUniverseService) UpdateAllCharactersESI(ctx context.Context) (set.Set[int32], error) {
	var changed set.Set[int32]
	ids, err := s.st.ListEveCharacterIDs(ctx)
	if err != nil {
		return changed, err
	}
	if ids.Size() == 0 {
		return changed, nil
	}
	ids2 := ids.Slice()
	hasChanged := make([]bool, len(ids2))
	g := new(errgroup.Group)
	for i, id := range ids2 {
		g.Go(func() error {
			changed, err := s.updateCharacterESI(ctx, id)
			if err != nil {
				return err
			}
			hasChanged[i] = changed
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return changed, err
	}
	for i, id := range ids2 {
		if hasChanged[i] {
			changed.Add(id)
		}
	}
	slog.Info("Finished updating eve characters", "count", ids.Size(), "changed", changed)
	return changed, nil
}

// updateCharacterESI updates a character from ESI and reports whether it has changed.
func (s *EveUniverseService) updateCharacterESI(ctx context.Context, characterID int32) (bool, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("updateCharacterESI-%d", characterID), func() (any, error) {
		c, err := s.st.GetEveCharacter(ctx, characterID)
		if err != nil {
			return false, err
		}
		old := *c
		// Fetch character
		o, r, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, c.ID, nil)
		if err != nil {
			if r != nil && r.StatusCode == http.StatusNotFound {
				s.st.DeleteEveCharacter(ctx, characterID)
				slog.Info("EVE Character no longer exists and was deleted", "characterID", characterID)
				return true, nil
			}
			return false, err
		}
		c.Name = o.Name
		c.Description = o.Description
		c.SecurityStatus = float64(o.SecurityStatus)
		c.Title = o.Title
		ids := set.Of(c.ID, o.CorporationId, o.AllianceId, o.FactionId)
		ids.Delete(0)
		m, err := s.ToEntities(ctx, ids)
		if err != nil {
			return false, err
		}
		c.Alliance = m[o.AllianceId]
		c.Corporation = m[o.CorporationId]
		c.Faction = m[o.FactionId]
		// Fetch affiliations
		affiliations, _, err := s.esiClient.ESI.CharacterApi.PostCharactersAffiliation(ctx, []int32{c.ID}, nil)
		if err != nil {
			return false, err
		}
		if len(affiliations) > 0 {
			x := affiliations[0]
			ids := set.Of(c.ID, x.CorporationId, x.AllianceId, x.FactionId)
			ids.Delete(0)
			m, err := s.ToEntities(ctx, ids)
			if err != nil {
				return false, err
			}
			c.Alliance = m[x.AllianceId]
			c.Corporation = m[x.CorporationId]
			c.Faction = m[x.FactionId]
		}
		// Update
		if err := s.st.UpdateEveCharacter(ctx, c); err != nil {
			return false, err
		}
		if _, err := s.st.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
			Category: app.EveEntityCharacter,
			ID:       characterID,
			Name:     c.Name,
		}); err != nil {
			return false, err
		}
		hasChanged := !old.Equal(*c)
		slog.Info("Updated eve character from ESI", "characterID", c.ID, "changed", hasChanged)
		return hasChanged, nil
	})
	if err != nil {
		return false, fmt.Errorf("updateCharacterESI: %w", err)
	}
	return x.(bool), nil
}
