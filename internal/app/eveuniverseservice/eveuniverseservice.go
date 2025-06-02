// Package eveuniverseservice contains EVE universe service.
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

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/dustin/go-humanize"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// EveUniverseService provides access to Eve Online models with on-demand loading from ESI and persistent local caching.
type EveUniverseService struct {
	// Now returns the current time in UTC. Can be overwritten for tests.
	Now func() time.Time

	esiClient *goesi.APIClient
	scs       *statuscacheservice.StatusCacheService
	sfg       *singleflight.Group
	st        *storage.Storage
}

type Params struct {
	ESIClient          *goesi.APIClient
	StatusCacheService *statuscacheservice.StatusCacheService
	Storage            *storage.Storage
}

// New returns a new instance of an Eve universe service.
func New(args Params) *EveUniverseService {
	eu := &EveUniverseService{
		scs:       args.StatusCacheService,
		esiClient: args.ESIClient,
		st:        args.Storage,
		sfg:       new(singleflight.Group),
		Now: func() time.Time {
			return time.Now().UTC()
		},
	}
	return eu
}

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

func (s *EveUniverseService) GetOrCreateCharacterESI(ctx context.Context, id int32) (*app.EveCharacter, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateCharacterESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveCharacter(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		r, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		ids := set.Of(id, r.CorporationId)
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
		if err := s.st.CreateEveCharacter(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Created eve character", "ID", id)
		return s.st.GetEveCharacter(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveCharacter), nil
}

// UpdateAllCharactersESI updates all known Eve characters from ESI.
func (s *EveUniverseService) UpdateAllCharactersESI(ctx context.Context) error {
	ids, err := s.st.ListEveCharacterIDs(ctx)
	if err != nil {
		return err
	}
	if ids.Size() == 0 {
		return nil
	}
	g := new(errgroup.Group)
	g.SetLimit(5)
	for id := range ids.All() {
		id := id
		g.Go(func() error {
			return s.updateCharacterESI(ctx, id)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	slog.Info("Finished updating eve characters", "count", ids.Size())
	return nil
}

func (s *EveUniverseService) updateCharacterESI(ctx context.Context, characterID int32) error {
	c, err := s.st.GetEveCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	// TODO: Refactor to use ToEntities()
	g := new(errgroup.Group)
	g.Go(func() error {
		rr, _, err := s.esiClient.ESI.CharacterApi.PostCharactersAffiliation(ctx, []int32{c.ID}, nil)
		if err != nil {
			return err
		}
		if len(rr) == 0 {
			return nil
		}
		r := rr[0]
		_, err = s.AddMissingEntities(ctx, set.Of(c.ID, r.CorporationId, r.AllianceId, r.FactionId))
		if err != nil {
			return err
		}
		corporation, err := s.st.GetEveEntity(ctx, r.CorporationId)
		if err != nil {
			return err
		}
		c.Corporation = corporation
		if r.AllianceId != 0 {
			alliance, err := s.st.GetEveEntity(ctx, r.AllianceId)
			if err != nil {
				return err
			}
			c.Alliance = alliance
		}
		if r.FactionId != 0 {
			faction, err := s.st.GetEveEntity(ctx, r.FactionId)
			if err != nil {
				return err
			}
			c.Faction = faction
		}
		return nil
	})
	g.Go(func() error {
		r2, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterId(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		c.Description = r2.Description
		c.SecurityStatus = float64(r2.SecurityStatus)
		c.Title = r2.Title
		return nil
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("update EveCharacter %d: %w", c.ID, err)
	}
	if err := s.st.UpdateEveCharacter(ctx, c); err != nil {
		return err
	}
	slog.Info("Updated eve character from ESI", "characterID", c.ID)
	return nil
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
		o, err := s.st.GetEveCorporation(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		r, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		ids := set.Of(id, r.CeoId, r.CreatorId, r.AllianceId, r.FactionId, r.HomeStationId)
		ids.DeleteFunc(func(id int32) bool {
			return id < 2
		})
		if _, err := s.AddMissingEntities(ctx, ids); err != nil {
			return nil, err
		}
		optionalFromSpecialEntityID := func(v int32) optional.Optional[int32] {
			if v == 0 || v == 1 {
				return optional.Optional[int32]{}
			}
			return optional.From(v)
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
	g.SetLimit(5)
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

func (s *EveUniverseService) GetDogmaAttribute(ctx context.Context, id int32) (*app.EveDogmaAttribute, error) {
	return s.st.GetEveDogmaAttribute(ctx, id)
}

func (s *EveUniverseService) GetOrCreateDogmaAttributeESI(ctx context.Context, id int32) (*app.EveDogmaAttribute, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("createDogmaAttributeFromESI-%d", id), func() (any, error) {
		o1, err := s.st.GetEveDogmaAttribute(ctx, id)
		if err == nil {
			return o1, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		d, _, err := s.esiClient.ESI.DogmaApi.GetDogmaAttributesAttributeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveDogmaAttributeParams{
			ID:           d.AttributeId,
			DefaultValue: d.DefaultValue,
			Description:  d.Description,
			DisplayName:  d.DisplayName,
			IconID:       d.IconId,
			Name:         d.Name,
			IsHighGood:   d.HighIsGood,
			IsPublished:  d.Published,
			IsStackable:  d.Stackable,
			UnitID:       app.EveUnitID(d.UnitId),
		}
		o2, err := s.st.CreateEveDogmaAttribute(ctx, arg)
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve dogma attribute", "ID", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveDogmaAttribute), nil
}

// FormatDogmaValue returns a formatted value.
func (s *EveUniverseService) FormatDogmaValue(ctx context.Context, value float32, unitID app.EveUnitID) (string, int32) {
	return formatDogmaValue(ctx, formatDogmaValueParams{
		value:                        value,
		unitID:                       unitID,
		getDogmaAttribute:            s.GetDogmaAttribute,
		getOrCreateDogmaAttributeESI: s.GetOrCreateDogmaAttributeESI,
		getType:                      s.GetType,
		getOrCreateTypeESI:           s.GetOrCreateTypeESI,
	})
}

type formatDogmaValueParams struct {
	value                        float32
	unitID                       app.EveUnitID
	getDogmaAttribute            func(context.Context, int32) (*app.EveDogmaAttribute, error)
	getOrCreateDogmaAttributeESI func(context.Context, int32) (*app.EveDogmaAttribute, error)
	getType                      func(context.Context, int32) (*app.EveType, error)
	getOrCreateTypeESI           func(context.Context, int32) (*app.EveType, error)
}

func formatDogmaValue(ctx context.Context, args formatDogmaValueParams) (string, int32) {
	defaultFormatter := func(v float32) string {
		return humanize.CommafWithDigits(float64(v), 2)
	}
	now := time.Now()
	value := args.value
	switch args.unitID {
	case app.EveUnitAbsolutePercent:
		return fmt.Sprintf("%.0f%%", value*100), 0
	case app.EveUnitAcceleration:
		return fmt.Sprintf("%s m/s²", defaultFormatter(value)), 0
	case app.EveUnitAttributeID:
		da, err := args.getDogmaAttribute(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := args.getOrCreateDogmaAttributeESI(ctx, int32(value))
				if err != nil {
					slog.Error("Failed to fetch dogma attribute from ESI", "ID", value, "err", err)
				}
			}()
			return "?", 0
		}
		return da.DisplayName, da.IconID
	case app.EveUnitAttributePoints:
		return fmt.Sprintf("%s points", defaultFormatter(value)), 0
	case app.EveUnitCapacitorUnits:
		return fmt.Sprintf("%s GJ", humanize.FormatFloat("#,###.#", float64(value))), 0
	case app.EveUnitDroneBandwidth:
		return fmt.Sprintf("%s Mbit/s", defaultFormatter(value)), 0
	case app.EveUnitHitpoints:
		return fmt.Sprintf("%s HP", defaultFormatter(value)), 0
	case app.EveUnitInverseAbsolutePercent:
		return fmt.Sprintf("%.0f%%", (1-value)*100), 0
	case app.EveUnitLength:
		if value > 1000 {
			return fmt.Sprintf("%s km", defaultFormatter(value/float32(1000))), 0
		} else {
			return fmt.Sprintf("%s m", defaultFormatter(value)), 0
		}
	case app.EveUnitLevel:
		return fmt.Sprintf("Level %s", defaultFormatter(value)), 0
	case app.EveUnitLightYear:
		return fmt.Sprintf("%.1f LY", value), 0
	case app.EveUnitMass:
		return fmt.Sprintf("%s kg", defaultFormatter(value)), 0
	case app.EveUnitMegaWatts:
		return fmt.Sprintf("%s MW", defaultFormatter(value)), 0
	case app.EveUnitMillimeters:
		return fmt.Sprintf("%s mm", defaultFormatter(value)), 0
	case app.EveUnitMilliseconds:
		return strings.TrimSpace(humanize.RelTime(now, now.Add(time.Duration(value)*time.Millisecond), "", "")), 0
	case app.EveUnitMultiplier:
		return fmt.Sprintf("%.3f x", value), 0
	case app.EveUnitPercentage:
		return fmt.Sprintf("%.0f%%", value*100), 0
	case app.EveUnitTeraflops:
		return fmt.Sprintf("%s tf", defaultFormatter(value)), 0
	case app.EveUnitVolume:
		return fmt.Sprintf("%s m3", defaultFormatter(value)), 0
	case app.EveUnitWarpSpeed:
		return fmt.Sprintf("%s AU/s", defaultFormatter(value)), 0
	case app.EveUnitTypeID:
		et, err := args.getType(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := args.getOrCreateTypeESI(ctx, int32(value))
				if err != nil {
					slog.Error("Failed to fetch type from ESI", "typeID", value, "err", err)
				}
			}()
			return "?", 0
		}
		return et.Name, et.IconID
	case app.EveUnitUnits:
		return fmt.Sprintf("%s units", defaultFormatter(value)), 0
	case app.EveUnitNone, app.EveUnitHardpoints, app.EveUnitFittingSlots, app.EveUnitSlot:
		return defaultFormatter(value), 0
	}
	return fmt.Sprintf("%s ???", defaultFormatter(value)), 0
}

func (s *EveUniverseService) UpdateShipSkills(ctx context.Context) error {
	return s.st.UpdateEveShipSkills(ctx)
}

func (s *EveUniverseService) ListTypeDogmaAttributesForType(
	ctx context.Context,
	typeID int32,
) ([]*app.EveTypeDogmaAttribute, error) {
	return s.st.ListEveTypeDogmaAttributesForType(ctx, typeID)
}

// TODO: Not fully thread safe: Might update for same ID multiple times.

// MarketPrice returns the average market price for a type. Or empty when no price is known for this type.
func (s *EveUniverseService) MarketPrice(ctx context.Context, typeID int32) (optional.Optional[float64], error) {
	var v optional.Optional[float64]
	o, err := s.st.GetEveMarketPrice(ctx, typeID)
	if errors.Is(err, app.ErrNotFound) {
		return v, nil
	} else if err != nil {
		return v, err
	}
	return optional.From(o.AveragePrice), nil
}

// TODO: Change to bulk create

func (s *EveUniverseService) updateMarketPricesESI(ctx context.Context) error {
	_, err, _ := s.sfg.Do("updateMarketPricesESI", func() (any, error) {
		prices, _, err := s.esiClient.ESI.MarketApi.GetMarketsPrices(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, p := range prices {
			arg := storage.UpdateOrCreateEveMarketPriceParams{
				TypeID:        p.TypeId,
				AdjustedPrice: p.AdjustedPrice,
				AveragePrice:  p.AveragePrice,
			}
			if err := s.st.UpdateOrCreateEveMarketPrice(ctx, arg); err != nil {
				return nil, err
			}
		}
		slog.Info("Updated market prices", "count", len(prices))
		return nil, nil
	})
	return err
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

// CharacterCorporationHistory returns a list of all the alliances a corporation has been a member of in descending order.
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

func (s *EveUniverseService) GetOrCreateRaceESI(ctx context.Context, id int32) (*app.EveRace, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateRaceESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveRace(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		races, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRaces(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, race := range races {
			if race.RaceId == id {
				arg := storage.CreateEveRaceParams{
					ID:          race.RaceId,
					Description: race.Description,
					Name:        race.Name,
				}
				o, err := s.st.CreateEveRace(ctx, arg)
				if err != nil {
					return nil, err
				}
				slog.Info("Created eve race", "id", id)
				return o, nil
			}
		}
		return nil, fmt.Errorf("race with ID %d not found: %w", id, app.ErrNotFound)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveRace), nil
}

func (s *EveUniverseService) GetOrCreateSchematicESI(ctx context.Context, id int32) (*app.EveSchematic, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateSchematicESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveSchematic(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		d, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetUniverseSchematicsSchematicId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveSchematicParams{
			ID:        id,
			CycleTime: int(d.CycleTime),
			Name:      d.SchematicName,
		}
		o2, err := s.st.CreateEveSchematic(ctx, arg)
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve schematic", "id", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveSchematic), nil
}

func (s *EveUniverseService) getSectionStatus(ctx context.Context, section app.GeneralSection) (*app.GeneralSectionStatus, error) {
	o, err := s.st.GetGeneralSectionStatus(ctx, section)
	if errors.Is(err, app.ErrNotFound) {
		return nil, nil
	}
	return o, err
}

func (s *EveUniverseService) UpdateSection(ctx context.Context, section app.GeneralSection, forceUpdate bool) (bool, error) {
	status, err := s.getSectionStatus(ctx, section)
	if err != nil {
		return false, err
	}
	if !forceUpdate && status != nil {
		if !status.HasError() && !status.IsExpired() {
			return false, nil
		}
	}
	var f func(context.Context) error
	switch section {
	case app.SectionEveTypes:
		f = s.updateCategories
	case app.SectionEveCharacters:
		f = s.UpdateAllCharactersESI
	case app.SectionEveCorporations:
		f = s.UpdateAllCorporationsESI
	case app.SectionEveMarketPrices:
		f = s.updateMarketPricesESI
	default:
		slog.Warn("encountered unknown section", "section", section)
	}
	_, err, _ = s.sfg.Do(fmt.Sprintf("update-general-section-%s", section), func() (any, error) {
		slog.Debug("Started updating eveuniverse section", "section", section)
		startedAt := optional.From(time.Now())
		arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   section,
			StartedAt: &startedAt,
		}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
		if err != nil {
			return false, err
		}
		s.scs.SetGeneralSection(o)
		err = f(ctx)
		slog.Debug("Finished updating eveuniverse section", "section", section)
		return nil, err
	})
	if err != nil {
		errorMessage := app.ErrorDisplay(err)
		startedAt := optional.Optional[time.Time]{}
		arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   section,
			Error:     &errorMessage,
			StartedAt: &startedAt,
		}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
		if err != nil {
			return false, err
		}
		s.scs.SetGeneralSection(o)
		return false, err
	}
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
		Section: section,

		Error:       &errorMessage,
		CompletedAt: &completedAt,
		StartedAt:   &startedAt2,
	}
	o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
	if err != nil {
		return false, err
	}
	s.scs.SetGeneralSection(o)
	return true, nil
}

func (s *EveUniverseService) updateCategories(ctx context.Context) error {
	g := new(errgroup.Group)
	g.Go(func() error {
		return s.UpdateCategoryWithChildrenESI(ctx, app.EveCategorySkill)
	})
	g.Go(func() error {
		return s.UpdateCategoryWithChildrenESI(ctx, app.EveCategoryShip)
	})
	if err := g.Wait(); err != nil {
		return err
	}
	if err := s.UpdateShipSkills(ctx); err != nil {
		return err
	}
	return nil
}
