package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
)

func (s *EveUniverseService) GetCharacterESI(ctx context.Context, characterID int64) (*app.EveCharacter, error) {
	return s.st.GetEveCharacter(ctx, characterID)
}

func (s *EveUniverseService) GetOrCreateCharacterESI(ctx context.Context, characterID int64) (*app.EveCharacter, bool, error) {
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
func (s *EveUniverseService) UpdateOrCreateCharacterESI(ctx context.Context, characterID int64) (*app.EveCharacter, bool, error) {
	c1, err := s.st.GetEveCharacter(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		c1 = nil
	} else if err != nil {
		return nil, false, err
	}
	c2, err, _ := xsingleflight.Do(&s.sfg, fmt.Sprintf("UpdateOrCreateCharacterESI-%d", characterID), func() (*app.EveCharacter, error) {
		ec, r, err := s.esiClient.CharacterAPI.GetCharactersCharacterId(ctx, characterID).Execute()
		if err != nil {
			if r != nil && r.StatusCode == http.StatusNotFound {
				return nil, app.ErrNotFound // character does not exist
			}
			return nil, err
		}
		_, err = s.GetOrCreateRaceESI(ctx, ec.RaceId)
		if err != nil {
			return nil, err
		}
		affiliations, _, err := s.esiClient.CharacterAPI.PostCharactersAffiliation(ctx).RequestBody([]int64{characterID}).Execute()
		if err != nil {
			return nil, err
		}
		if len(affiliations) != 1 {
			return nil, fmt.Errorf("affiliations mismatch")
		}
		af := affiliations[0]
		if af.CharacterId != characterID {
			slog.Warn("affiliations mismatch", "characterID", characterID, "affiliations", affiliations)
			return nil, nil // FIXME: Temporary workaround
		}
		arg := storage.CreateEveCharacterParams{
			AllianceID:     optional.FromPtr(af.AllianceId),
			Birthday:       ec.Birthday,
			CorporationID:  af.CorporationId,
			Description:    optional.FromPtr(ec.Description),
			FactionID:      optional.FromPtr(af.FactionId),
			Gender:         ec.Gender,
			ID:             characterID,
			Name:           ec.Name,
			RaceID:         ec.RaceId,
			SecurityStatus: optional.FromPtr(ec.SecurityStatus),
			Title:          optional.FromPtr(ec.Title),
		}
		ids := set.Of(characterID, arg.CorporationID)
		if af.AllianceId != nil {
			ids.Add(*af.AllianceId)
		}
		if af.FactionId != nil {
			ids.Add(*af.FactionId)
		}
		_, err = s.AddMissingEntities(ctx, ids)
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
	if c2 == nil {
		return nil, false, nil
	}
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

// UpdateAllCharactersESI updates all known Eve characters from ESI
// and returns the IDs of all changed characters.
func (s *EveUniverseService) UpdateAllCharactersESI(ctx context.Context) (set.Set[int64], error) {
	var changed set.Set[int64]
	ids, err := s.st.ListEveCharacterIDs(ctx)
	if err != nil {
		return changed, err
	}
	if ids.Size() == 0 {
		return changed, nil
	}
	ids2 := slices.Collect(ids.All())
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
