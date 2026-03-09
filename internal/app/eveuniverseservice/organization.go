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

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"github.com/icrowley/fake"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/evesde"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// FetchAlliance fetches an alliance from ESI and returns it.
func (s *EVEUniverseService) FetchAlliance(ctx context.Context, allianceID int64) (*app.EveAlliance, error) {
	a, _, err := s.esiClient.AllianceAPI.GetAlliancesAllianceId(ctx, allianceID).Execute()
	if err != nil {
		return nil, err
	}
	ids := set.Of(allianceID, a.CreatorCorporationId, a.CreatorId)
	if x := a.ExecutorCorporationId; x != nil {
		ids.Add(*x)
	}
	if x := a.FactionId; x != nil {
		ids.Add(*x)
	}
	ids.DeleteFunc(func(id int64) bool {
		return id < 2
	})
	eeMap, err := s.ToEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	maps.DeleteFunc(eeMap, func(id int64, o *app.EveEntity) bool {
		return !o.Category.IsKnown()
	})
	o := &app.EveAlliance{
		Creator:            eeMap[a.CreatorId],
		CreatorCorporation: eeMap[a.CreatorCorporationId],
		DateFounded:        a.DateFounded,
		ID:                 allianceID,
		Name:               a.Name,
		Ticker:             a.Ticker,
	}
	if x := a.ExecutorCorporationId; x != nil {
		o.ExecutorCorporation = optional.New(eeMap[*x])
	}
	if x := a.FactionId; x != nil {
		o.Faction = optional.New(eeMap[*x])
	}
	return o, nil
}

// FetchAllianceCorporations fetches the corporations for an alliance from ESI and returns them.
func (s *EVEUniverseService) FetchAllianceCorporations(ctx context.Context, allianceID int64) ([]*app.EveEntity, error) {
	ids, _, err := s.esiClient.AllianceAPI.GetAlliancesAllianceIdCorporations(ctx, allianceID).Execute()
	if err != nil {
		return nil, err
	}
	_, err = s.AddMissingEntities(ctx, set.Union(set.Collect(slices.Values(ids)), set.Of(allianceID)))
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
func (s *EVEUniverseService) RandomizeAllAllianceNames(ctx context.Context) error {
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

func (s *EVEUniverseService) GetCorporation(ctx context.Context, corporationID int64) (*app.EveCorporation, error) {
	return s.st.GetEveCorporation(ctx, corporationID)
}

func (s *EVEUniverseService) GetOrCreateCorporationESI(ctx context.Context, id int64) (*app.EveCorporation, error) {
	o, err := s.st.GetEveCorporation(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return s.UpdateOrCreateCorporationFromESI(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *EVEUniverseService) UpdateOrCreateCorporationFromESI(ctx context.Context, corporationID int64) (*app.EveCorporation, error) {
	o, err, _ := xsingleflight.Do(&s.sfg, fmt.Sprintf("UpdateOrCreateCorporationFromESI-%d", corporationID), func() (*app.EveCorporation, error) {
		r, _, err := s.esiClient.CorporationAPI.GetCorporationsCorporationId(ctx, corporationID).Execute()
		if err != nil {
			return nil, err
		}

		ceoID := optionalFromSpecialEntityID(r.CeoId)
		creatorID := optionalFromSpecialEntityID(r.CreatorId)
		var factionID optional.Optional[int64]
		if app.IsNPCCorporation(corporationID) {
			if id, ok := evesde.NPCCorporationFactionID(corporationID); ok {
				factionID = optional.New(id)
			}
		} else {
			factionID = optional.FromPtr(r.FactionId)
		}
		allianceID := optional.FromPtr(r.AllianceId)
		homeStationID := optional.FromPtr(r.HomeStationId)

		ids := set.Of(corporationID)
		for _, o := range []optional.Optional[int64]{allianceID, ceoID, creatorID, factionID, homeStationID} {
			if v, ok := o.Value(); ok {
				ids.Add(v)
			}
		}
		if _, err := s.AddMissingEntities(ctx, ids); err != nil {
			return nil, err
		}

		if err := s.st.UpdateOrCreateEveCorporation(ctx, storage.UpdateOrCreateEveCorporationParams{
			AllianceID:    allianceID,
			CeoID:         ceoID,
			CreatorID:     creatorID,
			FactionID:     factionID,
			DateFounded:   optional.FromPtr(r.DateFounded),
			Description:   optional.FromPtr(r.Description),
			HomeStationID: homeStationID,
			ID:            corporationID,
			MemberCount:   r.MemberCount,
			Name:          r.Name,
			Shares:        optional.FromPtr(r.Shares),
			TaxRate:       r.TaxRate,
			Ticker:        r.Ticker,
			URL:           optional.FromPtr(r.Url),
			WarEligible:   optional.FromPtr(r.WarEligible),
		}); err != nil {
			return nil, err
		}
		slog.Info("Stored updated eve corporation", "ID", corporationID)
		return s.st.GetEveCorporation(ctx, corporationID)
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}

func optionalFromSpecialEntityID(v int64) optional.Optional[int64] {
	if v == 0 || v == 1 {
		return optional.Optional[int64]{}
	}
	return optional.New(v)
}

// UpdateAllCorporationsESI updates all known corporations from ESI.
func (s *EVEUniverseService) UpdateAllCorporationsESI(ctx context.Context) (set.Set[int64], error) {
	var changed set.Set[int64]
	ids, err := s.st.ListEveCorporationIDs(ctx)
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
			c1, err := s.GetCorporation(ctx, id)
			if err != nil {
				return err
			}
			c2, err := s.UpdateOrCreateCorporationFromESI(ctx, id)
			if err != nil {
				return err
			}
			_, err = s.st.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
				ID:       id,
				Category: app.EveEntityCorporation,
				Name:     c2.Name,
			})
			hasChanged[i] = !c1.Equal(c2)
			return err
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
	slog.Info("Finished updating eve corporations", "count", ids.Size(), "changed", changed)
	return changed, nil
}

// RandomizeAllCorporationNames randomizes the names of all characters.
func (s *EVEUniverseService) RandomizeAllCorporationNames(ctx context.Context) error {
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
	return s.scs.UpdateCorporations(ctx, s.st)
}

// FetchCharacterCorporationHistory returns a list of all the corporations a character has been a member of in descending order.
func (s *EVEUniverseService) FetchCharacterCorporationHistory(ctx context.Context, characterID int64) ([]app.MembershipHistoryItem, error) {
	items, _, err := s.esiClient.CharacterAPI.GetCharactersCharacterIdCorporationhistory(ctx, characterID).Execute()
	if err != nil {
		return nil, err
	}
	items2 := make([]organizationHistoryItem, len(items))
	for i, it := range items {
		items2[i] = organizationHistoryItem{
			OrganizationID: optional.New(it.CorporationId),
			IsDeleted:      optional.FromPtr(it.IsDeleted),
			RecordID:       it.RecordId,
			StartDate:      it.StartDate,
		}
	}
	return s.makeMembershipHistory(ctx, items2)
}

// FetchCorporationAllianceHistory returns a list of all the alliances a corporation has been a member of in descending order.
func (s *EVEUniverseService) FetchCorporationAllianceHistory(ctx context.Context, corporationID int64) ([]app.MembershipHistoryItem, error) {
	items, _, err := s.esiClient.CorporationAPI.GetCorporationsCorporationIdAlliancehistory(ctx, corporationID).Execute()
	if err != nil {
		return nil, err
	}
	items2 := xslices.Map(items, func(x esi.CorporationsCorporationIdAlliancehistoryGetInner) organizationHistoryItem {
		return organizationHistoryItem{
			OrganizationID: optional.FromPtr(x.AllianceId),
			IsDeleted:      optional.FromPtr(x.IsDeleted),
			RecordID:       x.RecordId,
			StartDate:      x.StartDate,
		}
	})
	return s.makeMembershipHistory(ctx, items2)
}

type organizationHistoryItem struct {
	OrganizationID optional.Optional[int64]
	IsDeleted      optional.Optional[bool]
	RecordID       int64
	StartDate      time.Time
}

func (s *EVEUniverseService) makeMembershipHistory(ctx context.Context, items []organizationHistoryItem) ([]app.MembershipHistoryItem, error) {
	ids := set.Collect(xiter.Map(slices.Values(items), func(x organizationHistoryItem) int64 {
		return x.OrganizationID.ValueOrZero()
	}))
	ids.DeleteFunc(func(id int64) bool {
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
		var orig *app.EveEntity
		if v, ok := it.OrganizationID.Value(); ok {
			orig = eeMap[v]
		}
		oo[i] = app.MembershipHistoryItem{
			Days:         days,
			EndDate:      endDate,
			IsDeleted:    it.IsDeleted,
			IsOldest:     i == 0,
			RecordID:     it.RecordID,
			StartDate:    it.StartDate,
			Organization: orig,
		}
	}
	slices.SortFunc(oo, func(a, b app.MembershipHistoryItem) int {
		return -cmp.Compare(a.RecordID, b.RecordID)
	})
	return oo, nil
}
