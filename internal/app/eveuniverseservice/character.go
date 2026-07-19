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

func (s *EVEUniverseService) GetCharacterESI(ctx context.Context, characterID int64) (*app.EveCharacter, error) {
	return s.st.GetEveCharacter(ctx, characterID)
}

func (s *EVEUniverseService) GetOrCreateCharacterESI(ctx context.Context, characterID int64) (*app.EveCharacter, bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetOrCreateCharacterESI %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, false, wrapErr(app.ErrInvalid)
	}
	o, err := s.st.GetEveCharacter(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		return s.UpdateOrCreateCharacterESI(ctx, characterID)
	}
	if err != nil {
		return nil, false, wrapErr(err)
	}
	return o, false, nil
}

// UpdateOrCreateCharacterESI updates or create a character from ESI.
// Returns the changed character when it was changed and reports whether it was changed.
// Returns [app.ErrNotFound] when the character does not exist.
func (s *EVEUniverseService) UpdateOrCreateCharacterESI(ctx context.Context, characterID int64) (*app.EveCharacter, bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterESI %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, false, wrapErr(app.ErrInvalid)
	}
	c1, err := s.st.GetEveCharacter(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		c1 = nil
	} else if err != nil {
		return nil, false, wrapErr(err)
	}
	c2, err, _ := xsingleflight.Do(&s.sfg, fmt.Sprintf("UpdateOrCreateCharacterESI-%d", characterID), func() (*app.EveCharacter, error) {
		ec, r, err := s.esiClient.CharacterAPI.GetCharactersDetail(ctx, characterID).Execute()
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
		arg := storage.CreateEveCharacterParams{
			AllianceID:     optional.FromPtr(ec.AllianceId),
			Birthday:       ec.Birthday,
			CorporationID:  ec.CorporationId,
			Description:    optional.FromPtr(ec.Description),
			FactionID:      optional.FromPtr(ec.FactionId),
			Gender:         ec.Gender,
			ID:             characterID,
			Name:           ec.Name,
			RaceID:         ec.RaceId,
			SecurityStatus: optional.FromPtr(ec.SecurityStatus),
			Title:          optional.FromPtr(ec.CorporationTitle),
		}
		affiliationsOK, err := func() (bool, error) {
			affiliations, _, err := s.esiClient.CharacterAPI.PostCharactersAffiliation(ctx).RequestBody([]int64{characterID}).Execute()
			if err != nil {
				return false, err
			}
			if len(affiliations) != 1 {
				slog.Warn("ignoring unexpected affiliations response", "characterID", characterID, "affiliations", affiliations)
				return false, nil
			}
			af := affiliations[0]
			if af.CharacterId == characterID {
				arg.AllianceID = optional.FromPtr(af.AllianceId)
				arg.CorporationID = af.CorporationId
				arg.FactionID = optional.FromPtr(af.FactionId)
			} else {
				slog.Warn("ignoring affiliations mismatch", "characterID", characterID, "affiliations", affiliations)
				return false, nil
			}
			return true, nil
		}()
		if err != nil {
			return nil, err
		}
		optionalID := func(o optional.Optional[*app.EveEntity]) optional.Optional[int64] {
			v, ok := o.Value()
			if !ok {
				return optional.Optional[int64]{}
			}
			return optional.New(v.ID)
		}
		if !affiliationsOK && c1 != nil {
			// don't use affiliation info from character endpoint
			// for updating existing characters
			// when response from affiliation endpoint was invalid
			arg.AllianceID = optionalID(c1.Alliance)
			arg.FactionID = optionalID(c1.Faction)
			arg.CorporationID = c1.Corporation.ID
		}
		changed := c1 == nil ||
			optionalID(c1.Alliance) != arg.AllianceID ||
			c1.Corporation.ID != arg.CorporationID ||
			optionalID(c1.Faction) != arg.AllianceID ||
			c1.Description != arg.Description ||
			c1.Name != arg.Name ||
			c1.SecurityStatus != arg.SecurityStatus ||
			c1.Title != arg.Title
		if !changed {
			return nil, nil
		}
		_, err = s.AddMissingEntities(ctx, set.Of(
			characterID,
			arg.CorporationID,
			arg.AllianceID.ValueOrZero(),
			arg.FactionID.ValueOrZero()),
		)
		if err != nil {
			return nil, err
		}
		if err := s.st.UpdateOrCreateEveCharacter(ctx, arg); err != nil {
			return nil, err
		}
		if c1 != nil && c1.Name != arg.Name {
			_, err := s.st.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
				ID:       characterID,
				Name:     arg.Name,
				Category: app.EveEntityCharacter,
			})
			if err != nil {
				return nil, err
			}
		}
		c2, err := s.st.GetEveCharacter(ctx, characterID)
		if err != nil {
			return nil, err
		}
		return c2, nil
	})
	if err != nil {
		return nil, false, wrapErr(err)
	}
	if c2 != nil {
		slog.Info("Updated eve character", "ID", characterID)
		return c2, true, nil
	}
	if c1 != nil {
		return c1, false, nil
	}
	return nil, false, wrapErr(fmt.Errorf("no character to return")) // should be unreachable
}

// UpdateAllCharactersESI updates all known Eve characters from ESI
// and returns the IDs of all changed characters.
func (s *EVEUniverseService) UpdateAllCharactersESI(ctx context.Context) (set.Set[int64], error) {
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
				err := s.st.DeleteEveCharacter(ctx, id)
				if err != nil {
					slog.Warn("Deleting character that no longer exists", "characterID", id, "error", err)
				} else {
					slog.Info("EVE Character no longer exists and was deleted", "characterID", id)
					hasChanged[i] = true
				}
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
