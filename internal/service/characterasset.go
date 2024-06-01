package service

import (
	"context"
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/helper/goesi"
	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/optional"
)

func (s *Service) ListCharacterAssetsInShipHangar(characterID int32, locationID int64) ([]*model.CharacterAsset, error) {
	ctx := context.Background()
	return s.r.ListCharacterAssetsInShipHangar(ctx, characterID, locationID)
}

func (s *Service) ListCharacterAssetsInItemHangar(characterID int32, locationID int64) ([]*model.CharacterAsset, error) {
	ctx := context.Background()
	return s.r.ListCharacterAssetsInItemHangar(ctx, characterID, locationID)
}

func (s *Service) ListCharacterAssetLocations(characterID int32) ([]*model.CharacterAssetLocation, error) {
	ctx := context.Background()
	return s.r.ListCharacterAssetLocations(ctx, characterID)
}

type esiCharacterAssetPlus struct {
	esi.GetCharactersCharacterIdAssets200Ok
	Name string
}

func (s *Service) updateCharacterAssetsESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionAssets {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			assets, err := goesi.FetchFromESIWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdAssetsOpts{
						Page: optional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, characterID, arg)
				})
			if err != nil {
				return false, err
			}
			ids := make([]int64, len(assets))
			for i, a := range assets {
				ids[i] = a.ItemId
			}
			names, err := s.fetchCharacterAssetNamesESI(ctx, characterID, ids)
			if err != nil {
				return false, err
			}
			assetsPlus := make([]esiCharacterAssetPlus, len(assets))
			for i, a := range assets {
				o := esiCharacterAssetPlus{
					GetCharactersCharacterIdAssets200Ok: a,
					Name:                                names[a.ItemId],
				}
				assetsPlus[i] = o
			}
			return assetsPlus, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			assets := data.([]esiCharacterAssetPlus)
			typeIDs := make([]int32, 0)
			locationIDs := make([]int64, 0)
			for _, a := range assets {
				typeIDs = append(typeIDs, a.TypeId)
				if a.LocationFlag == "Hangar" {
					locationIDs = append(locationIDs, a.LocationId)
				}
			}
			missingLocationIDs, err := s.r.MissingLocations(ctx, locationIDs)
			if err != nil {
				return err
			}
			for _, id := range missingLocationIDs {
				_, err := s.getOrCreateLocationESI(ctx, id)
				if err != nil {
					return err
				}
			}
			if err := s.EveUniverse.AddMissingEveTypes(ctx, typeIDs); err != nil {
				return err
			}
			ids, err := s.r.ListCharacterAssetIDs(ctx, characterID)
			if err != nil {
				return err
			}
			existingIDs := set.NewFromSlice(ids)
			for _, a := range assets {
				if existingIDs.Has(a.ItemId) {
					arg := storage.UpdateCharacterAssetParams{
						CharacterID:  characterID,
						ItemID:       a.ItemId,
						LocationFlag: a.LocationFlag,
						LocationID:   a.LocationId,
						LocationType: a.LocationType,
						Name:         a.Name,
						Quantity:     a.Quantity,
					}
					if err := s.r.UpdateCharacterAsset(ctx, arg); err != nil {
						return err
					}
				} else {
					arg := storage.CreateCharacterAssetParams{
						CharacterID:     characterID,
						EveTypeID:       a.TypeId,
						IsBlueprintCopy: a.IsBlueprintCopy,
						IsSingleton:     a.IsSingleton,
						ItemID:          a.ItemId,
						LocationFlag:    a.LocationFlag,
						LocationID:      a.LocationId,
						LocationType:    a.LocationType,
						Name:            a.Name,
						Quantity:        a.Quantity,
					}
					if err := s.r.CreateCharacterAsset(ctx, arg); err != nil {
						return err
					}
				}
				itemIDs := make([]int64, len(assets))
				for i, a := range assets {
					itemIDs[i] = a.ItemId
				}
				if err := s.r.DeleteExcludedCharacterAssets(ctx, characterID, itemIDs); err != nil {
					return err
				}
			}
			return nil
		})
}

func (s *Service) fetchCharacterAssetNamesESI(ctx context.Context, characterID int32, ids []int64) (map[int64]string, error) {
	m := make(map[int64]string)
	for _, chunk := range chunkBy(ids, 1000) { // API allows 1000 IDs max
		names, _, err := s.esiClient.ESI.AssetsApi.PostCharactersCharacterIdAssetsNames(ctx, characterID, chunk, nil)
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			if n.Name != "None" {
				m[n.ItemId] = n.Name
			}
		}
	}
	return m, nil
}
