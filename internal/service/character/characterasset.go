package character

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

func (s *CharacterService) ListCharacterAssetsInShipHangar(ctx context.Context, characterID int32, locationID int64) ([]*model.CharacterAsset, error) {
	return s.st.ListCharacterAssetsInShipHangar(ctx, characterID, locationID)
}

func (s *CharacterService) ListCharacterAssetsInItemHangar(ctx context.Context, characterID int32, locationID int64) ([]*model.CharacterAsset, error) {
	return s.st.ListCharacterAssetsInItemHangar(ctx, characterID, locationID)
}

func (s *CharacterService) ListCharacterAssetsInLocation(ctx context.Context, characterID int32, locationID int64) ([]*model.CharacterAsset, error) {
	return s.st.ListCharacterAssetsInLocation(ctx, characterID, locationID)
}

func (s *CharacterService) ListCharacterAssetLocations(ctx context.Context, characterID int32) ([]*model.CharacterAssetLocation, error) {
	return s.st.ListCharacterAssetLocations(ctx, characterID)
}

func (s *CharacterService) ListAllCharacterAssets(ctx context.Context) ([]*model.CharacterAsset, error) {
	return s.st.ListAllCharacterAssets(ctx)
}

type esiCharacterAssetPlus struct {
	esi.GetCharactersCharacterIdAssets200Ok
	Name string
}

func (s *CharacterService) updateCharacterAssetsESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
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
			itemIDs := set.New[int64]()
			for _, ca := range assets {
				itemIDs.Add(ca.ItemId)
			}
			typeIDs := set.New[int32]()
			locationIDs := set.New[int64]()
			for _, ca := range assets {
				typeIDs.Add(ca.TypeId)
				if !itemIDs.Has(ca.LocationId) {
					locationIDs.Add(ca.LocationId) // location IDs that are not referencing other itemIDs are locations
				}
			}
			missingLocationIDs, err := s.st.MissingEveLocations(ctx, locationIDs.ToSlice())
			if err != nil {
				return err
			}
			for _, id := range missingLocationIDs {
				_, err := s.eu.GetOrCreateEveLocationESI(ctx, id)
				if err != nil {
					return err
				}
			}
			if err := s.eu.AddMissingEveTypes(ctx, typeIDs.ToSlice()); err != nil {
				return err
			}
			ids, err := s.st.ListCharacterAssetIDs(ctx, characterID)
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
					if err := s.st.UpdateCharacterAsset(ctx, arg); err != nil {
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
					if err := s.st.CreateCharacterAsset(ctx, arg); err != nil {
						return err
					}
				}
				if err := s.st.DeleteExcludedCharacterAssets(ctx, characterID, itemIDs.ToSlice()); err != nil {
					return err
				}
			}
			return nil
		})
}

func (s *CharacterService) fetchCharacterAssetNamesESI(ctx context.Context, characterID int32, ids []int64) (map[int64]string, error) {
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
