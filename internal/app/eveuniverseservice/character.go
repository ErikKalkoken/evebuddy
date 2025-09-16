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

func (s *EveUniverseService) GetOrCreateCharacterESI(ctx context.Context, characterID int32) (*app.EveCharacter, bool, error) {
	o, err := s.st.GetEveCharacter(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		return s.UpdateOrCreateCharacterESI(ctx, characterID)
	}
	if err != nil {
		return nil, false, err
	}
	return o, false, nil
}

// UpdateOrCreateCharacterESI updates or create a character from ESI.
// Returns [app.ErrNotFound] when the character does not exist.
func (s *EveUniverseService) UpdateOrCreateCharacterESI(ctx context.Context, characterID int32) (*app.EveCharacter, bool, error) {
	c1, err := s.st.GetEveCharacter(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		c1 = nil
	} else if err != nil {
		return nil, false, err
	}
	x, err, _ := s.sfg.Do(fmt.Sprintf("UpdateOrCreateCharacterESI-%d", characterID), func() (any, error) {
		ec, r, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, characterID, nil)
		if err != nil {
			if r != nil && r.StatusCode == http.StatusNotFound {
				return nil, app.ErrNotFound
			}
			return nil, err
		}
		_, err = s.GetOrCreateRaceESI(ctx, ec.RaceId)
		if err != nil {
			return nil, err
		}
		affiliations, _, err := s.esiClient.ESI.CharacterApi.PostCharactersAffiliation(ctx, []int32{characterID}, nil)
		if err != nil {
			return false, err
		}
		if len(affiliations) != 1 {
			return false, fmt.Errorf("affiliations mismatch")
		}
		af := affiliations[0]
		if af.CharacterId != characterID {
			return false, fmt.Errorf("wrong character %d in affiliations", af.CharacterId)
		}
		arg := storage.CreateEveCharacterParams{
			AllianceID:     af.AllianceId,
			Birthday:       ec.Birthday,
			CorporationID:  af.CorporationId,
			Description:    ec.Description,
			FactionID:      af.FactionId,
			Gender:         ec.Gender,
			ID:             characterID,
			Name:           ec.Name,
			RaceID:         ec.RaceId,
			SecurityStatus: float64(ec.SecurityStatus),
			Title:          ec.Title,
		}
		_, err = s.AddMissingEntities(ctx, set.Of(characterID, arg.CorporationID, arg.AllianceID, arg.FactionID))
		if err != nil {
			return nil, err
		}
		if err := s.st.UpdateOrCreateEveCharacter(ctx, arg); err != nil {
			return nil, err
		}
		c2, err := s.st.GetEveCharacter(ctx, characterID)
		if err != nil {
			return nil, err
		}
		return c2, nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("UpdateOrCreateCharacterESI %d: %w", characterID, err)
	}
	c2 := x.(*app.EveCharacter)
	changed := c1 == nil || c1.Hash() != c2.Hash()
	slog.Info("Updated eve character", "ID", characterID, "changed", changed)
	if c1 != nil && changed && c1.Name != c2.Name {
		_, err := s.st.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
			ID:       characterID,
			Name:     c2.Name,
			Category: app.EveEntityCharacter,
		})
		if err != nil {
			return nil, false, err
		}
	}
	return c2, changed, nil
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
	g.SetLimit(s.concurrencyLimit)
	for i, id := range ids2 {
		g.Go(func() error {
			_, changed, err := s.UpdateOrCreateCharacterESI(ctx, id)
			if errors.Is(err, app.ErrNotFound) {
				s.st.DeleteEveCharacter(ctx, id)
				slog.Info("EVE Character no longer exists and was deleted", "characterID", id)
				hasChanged[i] = true
				return nil
			}
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
