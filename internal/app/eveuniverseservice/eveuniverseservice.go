// Package eveuniverseservice contains EVE universe service.
package eveuniverseservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
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

// known invalid IDs
var invalidEveEntityIDs = []int32{
	1, // ID is used for fields, which are technically mandatory, but have no value (e.g. creator for NPC corps)
}

func (s *EveUniverseService) GetEntity(ctx context.Context, id int32) (*app.EveEntity, error) {
	return s.st.GetEveEntity(ctx, id)
}

// getValidEntity returns an EveEntity from storage for valid IDs and nil for invalid IDs.
func (s *EveUniverseService) getValidEntity(ctx context.Context, id int32) (*app.EveEntity, error) {
	if id == 0 || id == 1 {
		return nil, nil
	}
	return s.GetEntity(ctx, id)
}

func (s *EveUniverseService) GetOrCreateEntityESI(ctx context.Context, id int32) (*app.EveEntity, error) {
	o, err := s.st.GetEveEntity(ctx, id)
	if err == nil {
		return o, nil
	}
	if !errors.Is(err, app.ErrNotFound) {
		return nil, err
	}
	_, err = s.AddMissingEntities(ctx, set.Of(id))
	if err != nil {
		return nil, err
	}
	return s.st.GetEveEntity(ctx, id)
}

// ToEntities returns the resolved EveEntities for a list of valid entity IDs.
// It guarantees a result for every ID and will map unknown IDs (including 0 & 1) to empty EveEntity objects.
func (s *EveUniverseService) ToEntities(ctx context.Context, ids set.Set[int32]) (map[int32]*app.EveEntity, error) {
	r := make(map[int32]*app.EveEntity)
	if ids.Size() == 0 {
		return r, nil
	}
	ids2 := ids.Clone()
	ids2.Delete(0)
	if _, err := s.AddMissingEntities(ctx, ids2); err != nil {
		return nil, err
	}
	oo, err := s.st.ListEveEntitiesForIDs(ctx, ids2.Slice())
	if err != nil {
		return nil, err
	}
	for _, o := range oo {
		r[o.ID] = o
	}
	for id := range ids.All() {
		_, ok := r[id]
		if !ok {
			r[id] = &app.EveEntity{}
		}
	}
	return r, nil
}

// AddMissingEntities adds EveEntities from ESI for IDs missing in the database
// and returns which IDs where indeed missing.
//
// Invalid IDs (e.g. 0, 1) will be ignored.
func (s *EveUniverseService) AddMissingEntities(ctx context.Context, ids set.Set[int32]) (set.Set[int32], error) {
	if ids.Size() == 0 {
		return set.Set[int32]{}, nil
	}
	// Filter out known invalid IDs before continuing
	var bad, missing set.Set[int32]
	ids2 := ids.Clone()
	err := func() error {
		ids2.Delete(0) // do nothing with ID 0
		for _, id := range invalidEveEntityIDs {
			if ids2.Contains(id) {
				bad.Add(1)
				ids2.Delete(1)
			}
		}
		if ids2.Size() == 0 {
			return nil
		}
		// Identify missing IDs
		var err error
		missing, err = s.st.MissingEveEntityIDs(ctx, ids2)
		if err != nil {
			return err
		}
		if missing.Size() == 0 {
			return nil
		}
		// Call ESI to resolve missing IDs
		if missing.Size() > 0 {
			slog.Debug("Trying to resolve EveEntity IDs from ESI", "ids", missing)
		}
		var ee []esi.PostUniverseNames200Ok
		for chunk := range slices.Chunk(missing.Slice(), 1000) { // PostUniverseNames max is 1000 IDs
			eeChunk, badChunk, err := s.resolveIDsFromESI(ctx, chunk)
			if err != nil {
				return err
			}
			ee = append(ee, eeChunk...)
			bad.AddSeq(slices.Values(badChunk))
		}
		for _, entity := range ee {
			_, err := s.st.GetOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
				ID:       entity.Id,
				Name:     entity.Name,
				Category: eveEntityCategoryFromESICategory(entity.Category),
			},
			)
			if err != nil {
				return err
			}
		}
		slog.Info("Stored newly resolved EveEntities", "count", len(ee))
		return nil
	}()
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("AddMissingEntities: %w", err)
	}
	if bad.Size() > 0 {
		for id := range bad.All() {
			arg := storage.CreateEveEntityParams{
				ID:       id,
				Name:     "?",
				Category: app.EveEntityUnknown,
			}
			if _, err := s.st.GetOrCreateEveEntity(ctx, arg); err != nil {
				slog.Error("Failed to mark unresolvable EveEntity", "id", id, "error", err)
			}
		}
		slog.Warn("Marking unresolvable EveEntity IDs as unknown", "ids", bad)
	}
	return missing, nil
}

func (s *EveUniverseService) resolveIDsFromESI(ctx context.Context, ids []int32) ([]esi.PostUniverseNames200Ok, []int32, error) {
	slog.Debug("Trying to resolve IDs from ESI", "count", len(ids))
	ee, resp, err := s.esiClient.ESI.UniverseApi.PostUniverseNames(ctx, ids, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			if len(ids) == 1 {
				slog.Warn("found unresolvable ID", "id", ids)
				return []esi.PostUniverseNames200Ok{}, ids, nil
			}
			i := len(ids) / 2
			ee1, bad1, err := s.resolveIDsFromESI(ctx, ids[:i])
			if err != nil {
				return nil, nil, err
			}
			ee2, bad2, err := s.resolveIDsFromESI(ctx, ids[i:])
			if err != nil {
				return nil, nil, err
			}
			return slices.Concat(ee1, ee2), slices.Concat(bad1, bad2), nil
		}
		return nil, nil, err
	}
	return ee, []int32{}, nil
}

func (s *EveUniverseService) ListEntitiesByPartialName(ctx context.Context, partial string) ([]*app.EveEntity, error) {
	return s.st.ListEveEntitiesByPartialName(ctx, partial)
}
func (s *EveUniverseService) ListEntitiesForIDs(ctx context.Context, ids []int32) ([]*app.EveEntity, error) {
	return s.st.ListEveEntitiesForIDs(ctx, ids)
}

func eveEntityCategoryFromESICategory(c string) app.EveEntityCategory {
	categoryMap := map[string]app.EveEntityCategory{
		"alliance":       app.EveEntityAlliance,
		"character":      app.EveEntityCharacter,
		"corporation":    app.EveEntityCorporation,
		"constellation":  app.EveEntityConstellation,
		"faction":        app.EveEntityFaction,
		"inventory_type": app.EveEntityInventoryType,
		"mailing_list":   app.EveEntityMailList,
		"region":         app.EveEntityRegion,
		"solar_system":   app.EveEntitySolarSystem,
		"station":        app.EveEntityStation,
	}
	c2, ok := categoryMap[c]
	if !ok {
		return app.EveEntityUnknown
	}
	return c2
}

func (s *EveUniverseService) GetType(ctx context.Context, id int32) (*app.EveType, error) {
	return s.st.GetEveType(ctx, id)
}

func (s *EveUniverseService) GetOrCreateCategoryESI(ctx context.Context, id int32) (*app.EveCategory, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateCategoryESI-%d", id), func() (any, error) {
		o1, err := s.st.GetEveCategory(ctx, id)
		if err == nil {
			return o1, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveCategoryParams{
			ID:          id,
			Name:        r.Name,
			IsPublished: r.Published,
		}
		o2, err := s.st.CreateEveCategory(ctx, arg)
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve category", "ID", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveCategory), nil
}

func (s *EveUniverseService) GetOrCreateGroupESI(ctx context.Context, id int32) (*app.EveGroup, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateGroupESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveGroup(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		group, _, err := s.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		c, err := s.GetOrCreateCategoryESI(ctx, group.CategoryId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveGroupParams{
			ID:          id,
			Name:        group.Name,
			CategoryID:  c.ID,
			IsPublished: group.Published,
		}
		if err := s.st.CreateEveGroup(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Created eve group", "ID", id)
		return s.st.GetEveGroup(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveGroup), nil
}

func (s *EveUniverseService) GetOrCreateTypeESI(ctx context.Context, id int32) (*app.EveType, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateTypeESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveType(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		t, _, err := s.esiClient.ESI.UniverseApi.GetUniverseTypesTypeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		g, err := s.GetOrCreateGroupESI(ctx, t.GroupId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveTypeParams{
			ID:             id,
			GroupID:        g.ID,
			Capacity:       t.Capacity,
			Description:    t.Description,
			GraphicID:      t.GraphicId,
			IconID:         t.IconId,
			IsPublished:    t.Published,
			MarketGroupID:  t.MarketGroupId,
			Mass:           t.Mass,
			Name:           t.Name,
			PackagedVolume: t.PackagedVolume,
			PortionSize:    int(t.PortionSize),
			Radius:         t.Radius,
			Volume:         t.Volume,
		}
		if err := s.st.CreateEveType(ctx, arg); err != nil {
			return nil, err
		}
		for _, o := range t.DogmaAttributes {
			x, err := s.GetOrCreateDogmaAttributeESI(ctx, o.AttributeId)
			if err != nil {
				return nil, err
			}
			switch x.Unit {
			case app.EveUnitGroupID:
				go func(ctx context.Context, groupID int32) {
					_, err := s.GetOrCreateGroupESI(ctx, groupID)
					if err != nil {
						slog.Error("Failed to fetch eve group %d", "ID", groupID, "err", err)
					}
				}(ctx, int32(o.Value))
			case app.EveUnitTypeID:
				go func(ctx context.Context, typeID int32) {
					_, err := s.GetOrCreateTypeESI(ctx, typeID)
					if err != nil {
						slog.Error("Failed to fetch eve type %d", "ID", typeID, "err", err)
					}
				}(ctx, int32(o.Value))
			}
			arg := storage.CreateEveTypeDogmaAttributeParams{
				DogmaAttributeID: o.AttributeId,
				EveTypeID:        id,
				Value:            o.Value,
			}
			if err := s.st.CreateEveTypeDogmaAttribute(ctx, arg); err != nil {
				return nil, err
			}
		}
		for _, o := range t.DogmaEffects {
			arg := storage.CreateEveTypeDogmaEffectParams{
				DogmaEffectID: o.EffectId,
				EveTypeID:     id,
				IsDefault:     o.IsDefault,
			}
			if err := s.st.CreateEveTypeDogmaEffect(ctx, arg); err != nil {
				return nil, err
			}
		}
		slog.Info("Created eve type", "ID", id)
		return s.st.GetEveType(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveType), nil
}

// AddMissingTypes fetches missing typeIDs from ESI.
func (s *EveUniverseService) AddMissingTypes(ctx context.Context, ids set.Set[int32]) error {
	missing, err := s.st.MissingEveTypes(ctx, ids)
	if err != nil {
		return err
	}
	if missing.Size() == 0 {
		return nil
	}
	slog.Debug("Trying to fetch missing EveTypes from ESI", "count", missing.Size())
	g := new(errgroup.Group)
	for id := range missing.All() {
		g.Go(func() error {
			_, err := s.GetOrCreateTypeESI(ctx, id)
			return err
		})
	}
	return g.Wait()
}

func (s *EveUniverseService) UpdateCategoryWithChildrenESI(ctx context.Context, categoryID int32) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("UpdateCategoryWithChildrenESI-%d", categoryID), func() (any, error) {
		var typeIds set.Set[int32]
		_, err := s.GetOrCreateCategoryESI(ctx, categoryID)
		if err != nil {
			return nil, err
		}
		category, _, err := s.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, categoryID, nil)
		if err != nil {
			return nil, err
		}
		g := new(errgroup.Group)
		for _, id := range category.Groups {
			g.Go(func() error {
				_, err := s.GetOrCreateGroupESI(ctx, id)
				return err
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		groupTypes := make([][]int32, len(category.Groups))
		g = new(errgroup.Group)
		for i, id := range category.Groups {
			g.Go(func() error {
				group, _, err := s.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
				if err != nil {
					return err
				}
				groupTypes[i] = group.Types
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		for _, ids := range groupTypes {
			typeIds.AddSeq(slices.Values(ids))
		}
		if err := s.AddMissingTypes(ctx, typeIds); err != nil {
			return nil, err
		}
		slog.Info("Updated eve types", "categoryID", categoryID, "count", typeIds.Size())
		return nil, nil
	})
	if err != nil {
		return err
	}
	return nil
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

func (s *EveUniverseService) GetStationServicesESI(ctx context.Context, id int32) ([]string, error) {
	o, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStationsStationId(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	slices.Sort(o.Services)
	return o.Services, nil
}

func (s *EveUniverseService) GetLocation(ctx context.Context, id int64) (*app.EveLocation, error) {
	o, err := s.st.GetLocation(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return nil, app.ErrNotFound
	}
	return o, err
}

func (s *EveUniverseService) ListLocations(ctx context.Context) ([]*app.EveLocation, error) {
	return s.st.ListEveLocation(ctx)
}

// GetOrCreateLocationESI return a structure when it already exists
// or else tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context
func (s *EveUniverseService) GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	o, err := s.st.GetLocation(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return s.UpdateOrCreateLocationESI(ctx, id)
	}
	return o, err
}

// UpdateOrCreateLocationESI tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context when trying to fetch a structure.
func (s *EveUniverseService) UpdateOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	y, err, _ := s.sfg.Do(fmt.Sprintf("updateOrCreateLocationESI-%d", id), func() (any, error) {
		o, err := s.st.GetLocation(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		var arg storage.UpdateOrCreateLocationParams
		switch app.LocationVariantFromID(id) {
		case app.EveLocationUnknown:
			t, err := s.GetOrCreateTypeESI(ctx, app.EveTypeSolarSystem)
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:        id,
				EveTypeID: optional.From(t.ID),
			}
		case app.EveLocationAssetSafety:
			t, err := s.GetOrCreateTypeESI(ctx, app.EveTypeAssetSafetyWrap)
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:        id,
				EveTypeID: optional.From(t.ID),
			}
		case app.EveLocationSolarSystem:
			et, err := s.GetOrCreateTypeESI(ctx, app.EveTypeSolarSystem)
			if err != nil {
				return nil, err
			}
			es, err := s.GetOrCreateSolarSystemESI(ctx, int32(id))
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:               id,
				EveTypeID:        optional.From(et.ID),
				EveSolarSystemID: optional.From(es.ID),
			}
		case app.EveLocationStation:
			station, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStationsStationId(ctx, int32(id), nil)
			if err != nil {
				return nil, err
			}
			_, err = s.GetOrCreateSolarSystemESI(ctx, station.SystemId)
			if err != nil {
				return nil, err
			}
			_, err = s.GetOrCreateTypeESI(ctx, station.TypeId)
			if err != nil {
				return nil, err
			}
			arg.EveTypeID = optional.From(station.TypeId)
			arg = storage.UpdateOrCreateLocationParams{
				ID:               id,
				EveSolarSystemID: optional.From(station.SystemId),
				EveTypeID:        optional.From(station.TypeId),
				Name:             station.Name,
			}
			if station.Owner != 0 {
				_, err = s.AddMissingEntities(ctx, set.Of(station.Owner))
				if err != nil {
					return nil, err
				}
				arg.OwnerID = optional.From(station.Owner)
			}
		case app.EveLocationStructure:
			if ctx.Value(goesi.ContextAccessToken) == nil {
				return nil, fmt.Errorf("eve location: token not set for fetching structure: %d", id)
			}
			structure, r, err := s.esiClient.ESI.UniverseApi.GetUniverseStructuresStructureId(ctx, id, nil)
			if err != nil {
				if r != nil && r.StatusCode == http.StatusForbidden {
					arg = storage.UpdateOrCreateLocationParams{ID: id}
					break
				}
				return nil, err
			}
			_, err = s.GetOrCreateSolarSystemESI(ctx, structure.SolarSystemId)
			if err != nil {
				return nil, err
			}
			_, err = s.AddMissingEntities(ctx, set.Of(structure.OwnerId))
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:               id,
				EveSolarSystemID: optional.From(structure.SolarSystemId),
				Name:             structure.Name,
				OwnerID:          optional.From(structure.OwnerId),
			}
			if structure.TypeId != 0 {
				myType, err := s.GetOrCreateTypeESI(ctx, structure.TypeId)
				if err != nil {
					return nil, err
				}
				arg.EveTypeID = optional.From(myType.ID)
			}
		default:
			return nil, fmt.Errorf("eve location: invalid ID in update or create: %d", id)
		}
		arg.UpdatedAt = time.Now()
		if err := s.st.UpdateOrCreateEveLocation(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Stored updated eve location", "ID", id)
		return s.st.GetLocation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveLocation), nil
}

// AddMissingLocations adds missing EveLocations from ESI.
func (s *EveUniverseService) AddMissingLocations(ctx context.Context, ids set.Set[int64]) error {
	missing, err := s.st.MissingEveLocations(ctx, ids)
	if err != nil {
		return err
	}
	entities, err := s.EntityIDsFromLocationsESI(ctx, missing.Slice())
	if err != nil {
		return err
	}
	if _, err := s.AddMissingEntities(ctx, entities); err != nil {
		return err
	}
	g := new(errgroup.Group)
	for id := range missing.All() {
		g.Go(func() error {
			_, err := s.GetOrCreateLocationESI(ctx, id)
			return err
		})
	}
	return g.Wait()
}

// EntityIDsFromLocationsESI returns the EveEntity IDs in EveLocation ids from ESI.
// This methods allows bulk resolving EveEntities before fetching many new locations from ESI.
func (s *EveUniverseService) EntityIDsFromLocationsESI(ctx context.Context, ids []int64) (set.Set[int32], error) {
	if len(ids) == 0 {
		return set.Set[int32]{}, nil
	}
	for _, id := range ids {
		if app.LocationVariantFromID(id) == app.EveLocationStructure {
			if ctx.Value(goesi.ContextAccessToken) == nil {
				return set.Set[int32]{}, fmt.Errorf("eve location: token not set for fetching structure: %d", id)
			}
			break
		}
	}
	entityIDs := make([]int32, len(ids))
	g := new(errgroup.Group)
	for i, id := range ids {
		g.Go(func() error {
			switch app.LocationVariantFromID(id) {
			case app.EveLocationStation:
				station, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStationsStationId(ctx, int32(id), nil)
				if err != nil {
					return err
				}
				if x := station.Owner; x != 0 {
					entityIDs[i] = x
				}
			case app.EveLocationStructure:
				structure, r, err := s.esiClient.ESI.UniverseApi.GetUniverseStructuresStructureId(ctx, id, nil)
				if err != nil {
					if r != nil && r.StatusCode == http.StatusForbidden {
						return nil
					}
					return err
				}
				entityIDs[i] = structure.OwnerId
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return set.Set[int32]{}, err
	}
	r := set.Of(xslices.Filter(entityIDs, func(x int32) bool {
		return x != 0 && x != 1 && x != -1
	})...)
	return r, nil
}

func (s *EveUniverseService) GetStargateSolarSystemsESI(ctx context.Context, stargateIDs []int32) ([]*app.EveSolarSystem, error) {
	g := new(errgroup.Group)
	systemIDs := make([]int32, len(stargateIDs))
	for i, id := range stargateIDs {
		g.Go(func() error {
			x, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStargatesStargateId(ctx, id, nil)
			if err != nil {
				return err
			}
			systemIDs[i] = x.Destination.SystemId
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	g = new(errgroup.Group)
	systems := make([]*app.EveSolarSystem, len(systemIDs))
	for i, id := range systemIDs {
		g.Go(func() error {
			st, err := s.GetOrCreateSolarSystemESI(ctx, id)
			if err != nil {
				return err
			}
			systems[i] = st
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	slices.SortFunc(systems, func(a, b *app.EveSolarSystem) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return systems, nil
}

func (s *EveUniverseService) GetSolarSystemPlanets(ctx context.Context, planets []app.EveSolarSystemPlanet) ([]*app.EvePlanet, error) {
	oo := make([]*app.EvePlanet, len(planets))
	g := new(errgroup.Group)
	for i, p := range planets {
		g.Go(func() error {
			st, err := s.GetOrCreatePlanetESI(ctx, p.PlanetID)
			if err != nil {
				return err
			}
			oo[i] = st
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	slices.SortFunc(oo, func(a, b *app.EvePlanet) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return oo, nil
}

func (s *EveUniverseService) GetStarTypeID(ctx context.Context, id int32) (int32, error) {
	x2, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStarsStarId(ctx, id, nil)
	if err != nil {
		return 0, err
	}
	return x2.TypeId, nil
}

func (s *EveUniverseService) GetSolarSystemInfoESI(ctx context.Context, solarSystemID int32) (int32, []app.EveSolarSystemPlanet, []int32, []*app.EveEntity, []*app.EveLocation, error) {
	x, _, err := s.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, solarSystemID, nil)
	if err != nil {
		return 0, nil, nil, nil, nil, err
	}
	planets := xslices.Map(x.Planets, func(p esi.GetUniverseSystemsSystemIdPlanet) app.EveSolarSystemPlanet {
		return app.EveSolarSystemPlanet{
			AsteroidBeltIDs: p.AsteroidBelts,
			MoonIDs:         p.Moons,
			PlanetID:        p.PlanetId,
		}
	})
	ids := slices.Concat([]int32{solarSystemID, x.ConstellationId}, x.Stations)
	_, err = s.AddMissingEntities(ctx, set.Of(ids...))
	if err != nil {
		return 0, nil, nil, nil, nil, err
	}
	stations := make([]*app.EveEntity, len(x.Stations))
	for i, id := range x.Stations {
		st, err := s.getValidEntity(ctx, id)
		if err != nil {
			return 0, nil, nil, nil, nil, err
		}
		stations[i] = st
	}
	slices.SortFunc(stations, func(a, b *app.EveEntity) int {
		return a.Compare(b)
	})
	xx, err := s.st.ListEveLocationInSolarSystem(ctx, solarSystemID)
	if err != nil {
		return 0, nil, nil, nil, nil, err
	}
	structures := xslices.Filter(xx, func(x *app.EveLocation) bool {
		return x.Variant() == app.EveLocationStructure
	})
	return x.StarId, planets, x.Stargates, stations, structures, nil
}

func (s *EveUniverseService) GetRegionConstellationsESI(ctx context.Context, id int32) ([]*app.EveEntity, error) {
	region, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRegionsRegionId(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	xx, err := s.ToEntities(ctx, set.Of(region.Constellations...))
	if err != nil {
		return nil, err
	}
	oo := slices.Collect(maps.Values(xx))
	slices.SortFunc(oo, func(a, b *app.EveEntity) int {
		return a.Compare(b)
	})
	return oo, nil
}

func (s *EveUniverseService) GetConstellationSolarSystemsESI(ctx context.Context, id int32) ([]*app.EveSolarSystem, error) {
	o, _, err := s.esiClient.ESI.UniverseApi.GetUniverseConstellationsConstellationId(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	g := new(errgroup.Group)
	systems := make([]*app.EveSolarSystem, len(o.Systems))
	for i, id := range o.Systems {
		g.Go(func() error {
			st, err := s.GetOrCreateSolarSystemESI(ctx, id)
			if err != nil {
				return err
			}
			systems[i] = st
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	slices.SortFunc(systems, func(a, b *app.EveSolarSystem) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return systems, nil
}

func (s *EveUniverseService) GetOrCreateRegionESI(ctx context.Context, id int32) (*app.EveRegion, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateRegionESI-%d", id), func() (any, error) {
		o1, err := s.st.GetEveRegion(ctx, id)
		if err == nil {
			return o1, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		region, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRegionsRegionId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveRegionParams{
			ID:          region.RegionId,
			Description: region.Description,
			Name:        region.Name,
		}
		o2, err := s.st.CreateEveRegion(ctx, arg)
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve region", "ID", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveRegion), nil
}

func (s *EveUniverseService) GetOrCreateConstellationESI(ctx context.Context, id int32) (*app.EveConstellation, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateConstellationESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveConstellation(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		constellation, _, err := s.esiClient.ESI.UniverseApi.GetUniverseConstellationsConstellationId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		_, err = s.GetOrCreateRegionESI(ctx, constellation.RegionId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveConstellationParams{
			ID:       constellation.ConstellationId,
			RegionID: constellation.RegionId,
			Name:     constellation.Name,
		}
		if err := s.st.CreateEveConstellation(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Created eve constellation", "ID", id)
		return s.st.GetEveConstellation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveConstellation), nil
}

func (s *EveUniverseService) GetOrCreateSolarSystemESI(ctx context.Context, id int32) (*app.EveSolarSystem, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateSolarSystemESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveSolarSystem(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		system, _, err := s.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		constellation, err := s.GetOrCreateConstellationESI(ctx, system.ConstellationId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveSolarSystemParams{
			ID:              system.SystemId,
			ConstellationID: constellation.ID,
			Name:            system.Name,
			SecurityStatus:  system.SecurityStatus,
		}
		if err := s.st.CreateEveSolarSystem(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Created eve solar system", "ID", id)
		return s.st.GetEveSolarSystem(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveSolarSystem), nil
}

func (s *EveUniverseService) GetOrCreatePlanetESI(ctx context.Context, id int32) (*app.EvePlanet, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreatePlanetESI-%d", id), func() (any, error) {
		o, err := s.st.GetEvePlanet(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		planet, _, err := s.esiClient.ESI.UniverseApi.GetUniversePlanetsPlanetId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		system, err := s.GetOrCreateSolarSystemESI(ctx, planet.SystemId)
		if err != nil {
			return nil, err
		}
		type_, err := s.GetOrCreateTypeESI(ctx, planet.TypeId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEvePlanetParams{
			ID:            planet.PlanetId,
			Name:          planet.Name,
			SolarSystemID: system.ID,
			TypeID:        type_.ID,
		}
		if err := s.st.CreateEvePlanet(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Created eve planet", "ID", id)
		return s.st.GetEvePlanet(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EvePlanet), nil
}

func (s *EveUniverseService) GetOrCreateMoonESI(ctx context.Context, id int32) (*app.EveMoon, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateMoonESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveMoon(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		moon, _, err := s.esiClient.ESI.UniverseApi.GetUniverseMoonsMoonId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		system, err := s.GetOrCreateSolarSystemESI(ctx, moon.SystemId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveMoonParams{
			ID:            moon.MoonId,
			Name:          moon.Name,
			SolarSystemID: system.ID,
		}
		if err := s.st.CreateEveMoon(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Created eve moon", "ID", id)
		return s.st.GetEveMoon(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveMoon), nil
}

// FetchRoute fetches a route between two solar systems from ESi and returns it.
// When no route can be found it returns an empty slice.
func (s *EveUniverseService) FetchRoute(ctx context.Context, args app.EveRouteHeader) ([]*app.EveSolarSystem, error) {
	m := map[app.EveRoutePreference]string{
		app.RouteShortest: "shortest",
		app.RouteSecure:   "secure",
		app.RouteInsecure: "insecure",
	}
	flag, ok := m[args.Preference]
	if !ok {
		return nil, fmt.Errorf("FetchRoute: flag %s: %w", args.Preference, app.ErrInvalid)
	}
	if args.Destination == nil || args.Origin == nil {
		return nil, app.ErrInvalid
	}
	if args.Destination.ID == args.Origin.ID {
		return []*app.EveSolarSystem{args.Origin}, nil
	}
	if args.Destination.IsWormholeSpace() || args.Origin.IsWormholeSpace() {
		return []*app.EveSolarSystem{}, nil // no route possible
	}
	ids, r, err := s.esiClient.ESI.RoutesApi.GetRouteOriginDestination(ctx, args.Destination.ID, args.Origin.ID, &esi.GetRouteOriginDestinationOpts{
		Flag: esioptional.NewString(flag),
	})
	if err != nil {
		if r.StatusCode == 404 {
			return []*app.EveSolarSystem{}, nil // no route found
		}
		return nil, err
	}
	systems := make([]*app.EveSolarSystem, len(ids))
	g := new(errgroup.Group)
	for i, id := range ids {
		g.Go(func() error {
			system, err := s.GetOrCreateSolarSystemESI(ctx, id)
			if err != nil {
				return err
			}
			systems[i] = system
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return systems, nil
}

// FetchRoutes returns routes for one or multiple headers.
func (s *EveUniverseService) FetchRoutes(ctx context.Context, headers []app.EveRouteHeader) (map[app.EveRouteHeader][]*app.EveSolarSystem, error) {
	results := make([][]*app.EveSolarSystem, len(headers))
	g := new(errgroup.Group)
	for i, h := range headers {
		g.Go(func() error {
			route, err := s.FetchRoute(ctx, h)
			if err != nil {
				return err
			}
			results[i] = route
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	results2 := make(map[app.EveRouteHeader][]*app.EveSolarSystem)
	for i, h := range headers {
		results2[h] = results[i]
	}
	return results2, nil
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
