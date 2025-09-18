// Package corporationservice contains the corporation service.
package corporationservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type CharacterService interface {
	CharacterTokenForCorporation(ctx context.Context, corporationID int32, roles set.Set[app.Role], scopes set.Set[string], checkToken bool) (*app.CharacterToken, error)
}

// CorporationService provides access to all managed Eve Online corporations both online and from local storage.
type CorporationService struct {
	concurrencyLimit int
	cs               CharacterService
	esiClient        *goesi.APIClient
	eus              *eveuniverseservice.EveUniverseService
	httpClient       *http.Client
	scs              *statuscacheservice.StatusCacheService
	sfg              *singleflight.Group
	st               *storage.Storage
}

type Params struct {
	CharacterService   CharacterService
	ConcurrencyLimit   int // max number of concurrent Goroutines (per group)
	EveUniverseService *eveuniverseservice.EveUniverseService
	StatusCacheService *statuscacheservice.StatusCacheService
	Storage            *storage.Storage
	// optional
	HTTPClient *http.Client
	EsiClient  *goesi.APIClient
}

// New creates a new corporation service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(arg Params) *CorporationService {
	s := &CorporationService{
		concurrencyLimit: -1, // Default is no limit
		cs:               arg.CharacterService,
		eus:              arg.EveUniverseService,
		scs:              arg.StatusCacheService,
		st:               arg.Storage,
		sfg:              new(singleflight.Group),
	}
	if arg.HTTPClient == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = arg.HTTPClient
	}
	if arg.EsiClient == nil {
		s.esiClient = goesi.NewAPIClient(s.httpClient, "")
	} else {
		s.esiClient = arg.EsiClient
	}
	if arg.ConcurrencyLimit > 0 {
		s.concurrencyLimit = arg.ConcurrencyLimit
	}
	return s
}

// GetCorporation returns a corporation from storage.
// Returns [app.ErrNotFound] if the corporation does not exist.
func (s *CorporationService) GetCorporation(ctx context.Context, corporationID int32) (*app.Corporation, error) {
	return s.st.GetCorporation(ctx, corporationID)
}

// GetAnyCorporation returns a random corporation from storage.
// Returns [app.ErrNotFound] if no corporation is found.
func (s *CorporationService) GetAnyCorporation(ctx context.Context) (*app.Corporation, error) {
	return s.st.GetAnyCorporation(ctx)
}

// getOrCreateCorporation gets or creates a corporation and returns it.
// It returns the error [app.ErrNotFound] for NPC corporations
func (s *CorporationService) getOrCreateCorporation(ctx context.Context, corporationID int32) (*app.Corporation, error) {
	c, err := s.eus.GetOrCreateCorporationESI(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	if x := c.EveEntity().IsNPC(); x.IsEmpty() || x.ValueOrZero() {
		return nil, app.ErrNotFound
	}
	o, err := s.st.GetOrCreateCorporation(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	if err := s.scs.UpdateCorporations(ctx); err != nil {
		return nil, err
	}
	return o, nil
}

// HasCorporation reports whether we have access to a corporation via an owned character.
func (s *CorporationService) HasCorporation(ctx context.Context, corporationID int32) (bool, error) {
	if corporationID == 0 {
		return false, nil
	}
	ids, err := s.st.ListCorporationIDs(ctx)
	if err != nil {
		return false, err
	}
	return ids.Contains(corporationID), nil
}

// ListCorporationsShort returns all corporations in short form.
func (s *CorporationService) ListCorporationsShort(ctx context.Context) ([]*app.EntityShort[int32], error) {
	return s.st.ListCorporationsShort(ctx)
}

// ListPrivilegedCorporations returns all corporations with privileged access.
func (s *CorporationService) ListPrivilegedCorporations(ctx context.Context) ([]*app.EntityShort[int32], error) {
	var roles set.Set[app.Role]
	for _, s := range []app.CorporationSection{app.CorporationSectionWalletJournal(app.Division1)} {
		roles.AddSeq(s.Roles().All())
	}
	return s.st.ListPrivilegedCorporationsShort(ctx, roles)
}

// ListCorporationIDs returns all corporation IDs.
func (s *CorporationService) ListCorporationIDs(ctx context.Context) (set.Set[int32], error) {
	return s.st.ListCorporationIDs(ctx)
}

// UpdateCorporations removes all corporations which no longer have a user's character as member
// and creates missing corporations.
// It reports whether any corporation was removed or added.
func (s *CorporationService) UpdateCorporations(ctx context.Context) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCorporations: %w", err)
	}
	current, err := s.ListCorporationIDs(ctx)
	if err != nil {
		return false, wrapErr(err)
	}
	valid, err := s.st.ListCharacterCorporationIDs(ctx)
	if err != nil {
		return false, wrapErr(err)
	}
	obsolete := set.Difference(current, valid)
	if obsolete.Size() > 0 {
		for id := range obsolete.All() {
			if err := s.st.DeleteCorporation(ctx, id); err != nil {
				return false, wrapErr(err)
			}
		}
		slog.Info("Deleted obsolete corporations", "corporationIDs", obsolete)
	}
	missing := set.Difference(valid, current)
	if missing.Size() > 0 {
		for id := range missing.All() {
			_, err := s.getOrCreateCorporation(ctx, id)
			if errors.Is(err, app.ErrNotFound) {
				continue
			}
			if err != nil {
				return false, wrapErr(err)
			}
		}
		slog.Info("Added missing corporations", "corporationIDs", missing)
	}
	return missing.Size() > 0 || obsolete.Size() > 0, nil
}

func (s *CorporationService) updateDivisionsESI(ctx context.Context, arg app.CorporationSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCorporationDivisions {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
			divisions, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationIdDivisions(ctx, arg.CorporationID, nil)
			if err != nil {
				return false, err
			}
			return divisions, nil
		},
		func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
			divisions := data.(esi.GetCorporationsCorporationIdDivisionsOk)
			for _, w := range divisions.Hangar {
				if err := s.st.UpdateOrCreateCorporationHangarName(ctx, storage.UpdateOrCreateCorporationHangarNameParams{
					CorporationID: arg.CorporationID,
					DivisionID:    w.Division,
					Name:          w.Name,
				}); err != nil {
					return err
				}
			}
			slog.Info("Updated corporation hangar names", "corporationID", arg.CorporationID)
			for _, w := range divisions.Wallet {
				if err := s.st.UpdateOrCreateCorporationWalletName(ctx, storage.UpdateOrCreateCorporationWalletNameParams{
					CorporationID: arg.CorporationID,
					DivisionID:    w.Division,
					Name:          w.Name,
				}); err != nil {
					return err
				}
			}
			slog.Info("Updated corporation wallet names", "corporationID", arg.CorporationID)
			return nil
		})
}
