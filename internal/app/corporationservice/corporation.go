// Package corporationservice contains the corporation service.
package corporationservice

import (
	"context"
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
	ValidCharacterTokenForCorporation(ctx context.Context, corporationID int32, roles set.Set[app.Role], scopes set.Set[string]) (*app.CharacterToken, error)
}

// CorporationService provides access to all managed Eve Online corporations both online and from local storage.
type CorporationService struct {
	cs         CharacterService
	esiClient  *goesi.APIClient
	eus        *eveuniverseservice.EveUniverseService
	httpClient *http.Client
	scs        *statuscacheservice.StatusCacheService
	sfg        *singleflight.Group
	st         *storage.Storage
}

type Params struct {
	CharacterService   CharacterService
	EveUniverseService *eveuniverseservice.EveUniverseService
	StatusCacheService *statuscacheservice.StatusCacheService
	Storage            *storage.Storage
	// optional
	HTTPClient *http.Client
	EsiClient  *goesi.APIClient
}

// New creates a new corporation service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(args Params) *CorporationService {
	s := &CorporationService{
		cs:  args.CharacterService,
		eus: args.EveUniverseService,
		scs: args.StatusCacheService,
		st:  args.Storage,
		sfg: new(singleflight.Group),
	}
	if args.HTTPClient == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = args.HTTPClient
	}
	if args.EsiClient == nil {
		s.esiClient = goesi.NewAPIClient(s.httpClient, "")
	} else {
		s.esiClient = args.EsiClient
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

func (s *CorporationService) GetOrCreateCorporation(ctx context.Context, corporationID int32) (*app.Corporation, error) {
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

// ListCorporationIDs returns all corporation IDs.
func (s *CorporationService) ListCorporationIDs(ctx context.Context) (set.Set[int32], error) {
	return s.st.ListCorporationIDs(ctx)
}

func (s *CorporationService) updateDivisionsESI(ctx context.Context, arg app.CorporationUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCorporationDivisions {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationUpdateSectionParams) (any, error) {
			divisions, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationIdDivisions(ctx, arg.CorporationID, nil)
			if err != nil {
				return false, err
			}
			return divisions, nil
		},
		func(ctx context.Context, arg app.CorporationUpdateSectionParams, data any) error {
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
