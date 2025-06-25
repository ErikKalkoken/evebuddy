// Package corporationservice contains the corporation service.
package corporationservice

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type CharacterService interface {
	ValidCharacterTokenForCorporation(ctx context.Context, corporationID int32, role app.Role) (*app.CharacterToken, error)
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

// UpdateSectionIfNeeded updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *CorporationService) UpdateSectionIfNeeded(ctx context.Context, arg app.CorporationUpdateSectionParams) (bool, error) {
	if arg.CorporationID == 0 || arg.Section == "" {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	if !arg.ForceUpdate {
		status, err := s.st.GetCorporationSectionStatus(ctx, arg.CorporationID, arg.Section)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				return false, err
			}
		} else {
			if !status.HasError() && !status.IsExpired() {
				return false, nil
			}
		}
	}
	var f func(context.Context, app.CorporationUpdateSectionParams) (bool, error)
	switch arg.Section {
	case app.SectionCorporationDivisions:
		f = s.updateDivisionsESI
	case app.SectionCorporationIndustryJobs:
		f = s.updateIndustryJobsESI
	case app.SectionCorporationWalletBalances:
		f = s.updateWalletBalancesESI
	case
		app.SectionCorporationWalletJournal1,
		app.SectionCorporationWalletJournal2,
		app.SectionCorporationWalletJournal3,
		app.SectionCorporationWalletJournal4,
		app.SectionCorporationWalletJournal5,
		app.SectionCorporationWalletJournal6,
		app.SectionCorporationWalletJournal7:
		f = s.updateWalletJournalESI
	case
		app.SectionCorporationWalletTransactions1,
		app.SectionCorporationWalletTransactions2,
		app.SectionCorporationWalletTransactions3,
		app.SectionCorporationWalletTransactions4,
		app.SectionCorporationWalletTransactions5,
		app.SectionCorporationWalletTransactions6,
		app.SectionCorporationWalletTransactions7:
		f = s.updateWalletTransactionESI
	default:
		return false, fmt.Errorf("update section: unknown section: %s", arg.Section)
	}
	key := fmt.Sprintf("update-corporation-section-%s-%d", arg.Section, arg.CorporationID)
	x, err, _ := s.sfg.Do(key, func() (any, error) {
		return f(ctx, arg)
	})
	if err != nil {
		errorMessage := err.Error()
		startedAt := optional.Optional[time.Time]{}
		arg2 := storage.UpdateOrCreateCorporationSectionStatusParams{
			CorporationID: arg.CorporationID,
			Section:       arg.Section,
			ErrorMessage:  &errorMessage,
			StartedAt:     &startedAt,
		}
		o, err2 := s.st.UpdateOrCreateCorporationSectionStatus(ctx, arg2)
		if err2 != nil {
			slog.Error("record error for failed section update: %s", "error", err2)
		}
		s.scs.SetCorporationSection(o)
		return false, fmt.Errorf("update corporation section from ESI for %v: %w", arg, err)
	}
	changed := x.(bool)
	slog.Info("Corporation section update completed", "corporationID", arg.CorporationID, "section", arg.Section, "forced", arg.ForceUpdate, "changed", changed)
	return changed, err
}

// updateSectionIfChanged updates a character section if it has changed
// and reports whether it has changed
func (s *CorporationService) updateSectionIfChanged(
	ctx context.Context,
	arg app.CorporationUpdateSectionParams,
	fetch func(ctx context.Context, arg app.CorporationUpdateSectionParams) (any, error),
	update func(ctx context.Context, arg app.CorporationUpdateSectionParams, data any) error,
) (bool, error) {
	startedAt := optional.From(time.Now())
	arg2 := storage.UpdateOrCreateCorporationSectionStatusParams{
		CorporationID: arg.CorporationID,
		Section:       arg.Section,
		StartedAt:     &startedAt,
	}
	o, err := s.st.UpdateOrCreateCorporationSectionStatus(ctx, arg2)
	if err != nil {
		return false, err
	}
	s.scs.SetCorporationSection(o)
	var hash, comment string
	var hasChanged bool
	token, err := s.cs.ValidCharacterTokenForCorporation(ctx, arg.CorporationID, arg.Section.Role())
	if errors.Is(err, app.ErrNotFound) {
		msg := "update skipped due to missing corporation member with required role"
		comment = msg + ": " + arg.Section.Role().Display()
		slog.Info("Section "+comment, "corporationID", arg.CorporationID, "section", arg.Section, "role", arg.Section.Role())
	} else if err != nil {
		return false, err
	} else {
		ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
		data, err := fetch(ctx, arg)
		if err != nil {
			return false, err
		}
		h, err := calcContentHash(data)
		if err != nil {
			return false, err
		}
		hash = h

		// identify if changed
		var notFound bool
		u, err := s.st.GetCorporationSectionStatus(ctx, arg.CorporationID, arg.Section)
		if errors.Is(err, app.ErrNotFound) {
			notFound = true
		} else if err != nil {
			return false, err
		}

		// update if needed
		hasChanged = u.ContentHash != hash
		if arg.ForceUpdate || notFound || hasChanged {
			if err := update(ctx, arg, data); err != nil {
				return false, err
			}
		}
	}

	// record completion
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	arg2 = storage.UpdateOrCreateCorporationSectionStatusParams{
		Comment:       &comment,
		CompletedAt:   &completedAt,
		ContentHash:   &hash,
		CorporationID: arg.CorporationID,
		ErrorMessage:  &errorMessage,
		Section:       arg.Section,
		StartedAt:     &startedAt2,
	}
	o, err = s.st.UpdateOrCreateCorporationSectionStatus(ctx, arg2)
	if err != nil {
		return false, err
	}
	s.scs.SetCorporationSection(o)
	slog.Debug("Has section changed", "corporationID", arg.CorporationID, "section", arg.Section, "changed", hasChanged)
	return hasChanged, nil
}

func calcContentHash(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b2 := md5.Sum(b)
	hash := hex.EncodeToString(b2[:])
	return hash, nil
}
