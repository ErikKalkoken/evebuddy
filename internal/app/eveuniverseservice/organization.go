package eveuniverseservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/antihax/goesi/esi"
	"github.com/icrowley/fake"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// FetchAlliance fetches an alliance from ESI and returns it.
func (s *EveUniverseService) FetchAlliance(ctx context.Context, allianceID int32) (*app.EveAlliance, error) {
	a, _, err := s.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(ctx, allianceID, nil)
	if err != nil {
		return nil, err
	}
	ids := set.Of(allianceID, a.CreatorCorporationId, a.CreatorId, a.ExecutorCorporationId, a.FactionId)
	ids.DeleteFunc(func(id int32) bool {
		return id < 2
	})
	eeMap, err := s.ToEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	maps.DeleteFunc(eeMap, func(id int32, o *app.EveEntity) bool {
		return !o.Category.IsKnown()
	})
	o := &app.EveAlliance{
		Creator:             eeMap[a.CreatorId],
		CreatorCorporation:  eeMap[a.CreatorCorporationId],
		DateFounded:         a.DateFounded,
		ExecutorCorporation: eeMap[a.ExecutorCorporationId],
		Faction:             eeMap[a.FactionId],
		ID:                  allianceID,
		Name:                a.Name,
		Ticker:              a.Ticker,
	}
	return o, nil
}

// FetchAllianceCorporations fetches the corporations for an alliance from ESI and returns them.
func (s *EveUniverseService) FetchAllianceCorporations(ctx context.Context, allianceID int32) ([]*app.EveEntity, error) {
	ids, _, err := s.esiClient.ESI.AllianceApi.GetAlliancesAllianceIdCorporations(ctx, allianceID, nil)
	if err != nil {
		return nil, err
	}
	_, err = s.AddMissingEntities(ctx, set.Union(set.Of(ids...), set.Of(allianceID)))
	if err != nil {
		return nil, err
	}
	oo, err := s.st.ListEveEntitiesForIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(oo, func(a, b *app.EveEntity) int {
		return strings.Compare(a.Name, b.Name)
	})
	return oo, nil
}

// RandomizeAllAllianceNames randomizes the names of all alliances.
func (s *EveUniverseService) RandomizeAllAllianceNames(ctx context.Context) error {
	ee, err := s.st.ListEveEntities(ctx)
	if err != nil {
		return err
	}
	alliances := xslices.Filter(ee, func(x *app.EveEntity) bool {
		return x.Category == app.EveEntityAlliance
	})
	for _, alliance := range alliances {
		name := fake.Company()
		err = s.updateEntityNameIfExists(ctx, alliance.ID, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *EveUniverseService) GetEveCorporation(ctx context.Context, corporationID int32) (*app.EveCorporation, error) {
	return s.st.GetEveCorporation(ctx, corporationID)
}

func (s *EveUniverseService) GetOrCreateCorporationESI(ctx context.Context, id int32) (*app.EveCorporation, error) {
	o, err := s.st.GetEveCorporation(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return s.UpdateOrCreateCorporationFromESI(ctx, id)
	}
	return o, err
}

func (s *EveUniverseService) UpdateOrCreateCorporationFromESI(ctx context.Context, id int32) (*app.EveCorporation, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("UpdateOrCreateCorporationFromESI-%d", id), func() (any, error) {
		r, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		ids := set.Of(id, r.AllianceId, r.CeoId, r.CreatorId, r.FactionId, r.HomeStationId)
		if _, err := s.AddMissingEntities(ctx, ids); err != nil {
			return nil, err
		}
		optionalFromSpecialEntityID := func(v int32) optional.Optional[int32] {
			if v == 0 || v == 1 {
				return optional.Optional[int32]{}
			}
			return optional.New(v)
		}
		arg := storage.UpdateOrCreateEveCorporationParams{
			AllianceID:    optional.FromIntegerWithZero(r.AllianceId),
			CeoID:         optionalFromSpecialEntityID(r.CeoId),
			CreatorID:     optionalFromSpecialEntityID(r.CreatorId),
			FactionID:     optional.FromIntegerWithZero(r.FactionId),
			DateFounded:   optional.FromTimeWithZero(r.DateFounded),
			Description:   r.Description,
			HomeStationID: optional.FromIntegerWithZero(r.HomeStationId),
			ID:            id,
			MemberCount:   r.MemberCount,
			Name:          r.Name,
			Shares:        optional.FromIntegerWithZero(r.Shares),
			TaxRate:       r.TaxRate,
			Ticker:        r.Ticker,
			URL:           r.Url,
			WarEligible:   r.WarEligible,
		}
		if err := s.st.UpdateOrCreateEveCorporation(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Stored updated eve corporation", "ID", id)
		return s.st.GetEveCorporation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveCorporation), nil
}

// UpdateAllCorporationsESI updates all known corporations from ESI.
func (s *EveUniverseService) UpdateAllCorporationsESI(ctx context.Context) error {
	ids, err := s.st.ListEveCorporationIDs(ctx)
	if err != nil {
		return err
	}
	if ids.Size() == 0 {
		return nil
	}
	g := new(errgroup.Group)
	for id := range ids.All() {
		g.Go(func() error {
			_, err := s.UpdateOrCreateCorporationFromESI(ctx, id)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	slog.Info("Finished updating eve corporations", "count", ids.Size())
	return nil
}

// RandomizeAllCorporationNames randomizes the names of all characters.
func (s *EveUniverseService) RandomizeAllCorporationNames(ctx context.Context) error {
	ids, err := s.st.ListEveCorporationIDs(ctx)
	if err != nil {
		return err
	}
	if ids.Size() == 0 {
		return nil
	}
	for id := range ids.All() {
		name := fake.Company()
		err := s.st.UpdateEveCorporationName(ctx, id, name)
		if err != nil {
			return err
		}
		err = s.updateEntityNameIfExists(ctx, id, name)
		if err != nil {
			return err
		}

	}
	return s.scs.UpdateCorporations(ctx)
}

// FetchCharacterCorporationHistory returns a list of all the corporations a character has been a member of in descending order.
func (s *EveUniverseService) FetchCharacterCorporationHistory(ctx context.Context, characterID int32) ([]app.MembershipHistoryItem, error) {
	items, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterIdCorporationhistory(ctx, characterID, nil)
	if err != nil {
		return nil, err
	}
	items2 := make([]organizationHistoryItem, len(items))
	for i, it := range items {
		items2[i] = organizationHistoryItem{
			OrganizationID: it.CorporationId,
			IsDeleted:      it.IsDeleted,
			RecordID:       int(it.RecordId),
			StartDate:      it.StartDate,
		}
	}
	return s.makeMembershipHistory(ctx, items2)
}

// FetchCorporationAllianceHistory returns a list of all the alliances a corporation has been a member of in descending order.
func (s *EveUniverseService) FetchCorporationAllianceHistory(ctx context.Context, corporationID int32) ([]app.MembershipHistoryItem, error) {
	items, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationIdAlliancehistory(ctx, corporationID, nil)
	if err != nil {
		return nil, err
	}
	items2 := xslices.Map(items, func(x esi.GetCorporationsCorporationIdAlliancehistory200Ok) organizationHistoryItem {
		return organizationHistoryItem{
			OrganizationID: x.AllianceId,
			IsDeleted:      x.IsDeleted,
			RecordID:       int(x.RecordId),
			StartDate:      x.StartDate,
		}
	})
	return s.makeMembershipHistory(ctx, items2)
}

type organizationHistoryItem struct {
	OrganizationID int32
	IsDeleted      bool
	RecordID       int
	StartDate      time.Time
}

func (s *EveUniverseService) makeMembershipHistory(ctx context.Context, items []organizationHistoryItem) ([]app.MembershipHistoryItem, error) {
	ids := set.Collect(xiter.Map(slices.Values(items), func(x organizationHistoryItem) int32 {
		return x.OrganizationID
	}))
	ids.DeleteFunc(func(id int32) bool {
		return id < 2
	})
	eeMap, err := s.ToEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b organizationHistoryItem) int {
		return cmp.Compare(a.RecordID, b.RecordID)
	})

	oo := make([]app.MembershipHistoryItem, len(items))
	for i, it := range items {
		var endDate time.Time
		if i+1 < len(items) {
			endDate = items[i+1].StartDate
		}
		var endDate2 time.Time
		if !endDate.IsZero() {
			endDate2 = endDate
		} else {
			endDate2 = s.Now()
		}
		days := int(endDate2.Sub(it.StartDate) / (time.Hour * 24))
		oo[i] = app.MembershipHistoryItem{
			Days:         days,
			EndDate:      endDate,
			IsDeleted:    it.IsDeleted,
			IsOldest:     i == 0,
			RecordID:     it.RecordID,
			StartDate:    it.StartDate,
			Organization: eeMap[it.OrganizationID],
		}
	}
	slices.SortFunc(oo, func(a, b app.MembershipHistoryItem) int {
		return -cmp.Compare(a.RecordID, b.RecordID)
	})
	return oo, nil
}
