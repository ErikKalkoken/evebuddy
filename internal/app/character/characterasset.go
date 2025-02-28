package character

import (
	"context"
	"log/slog"
	"net/http"
	"slices"

	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

const (
	assetNamesMaxIDs = 999
)

func (s *CharacterService) ListCharacterAssetsInShipHangar(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssetsInShipHangar(ctx, characterID, locationID)
}

func (s *CharacterService) ListCharacterAssetsInItemHangar(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssetsInItemHangar(ctx, characterID, locationID)
}

func (s *CharacterService) ListCharacterAssetsInLocation(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssetsInLocation(ctx, characterID, locationID)
}

func (s *CharacterService) ListCharacterAssets(ctx context.Context, characterID int32) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssets(ctx, characterID)
}

func (s *CharacterService) ListAllCharacterAssets(ctx context.Context) ([]*app.CharacterAsset, error) {
	return s.st.ListAllCharacterAssets(ctx)
}

type esiCharacterAssetPlus struct {
	esi.GetCharactersCharacterIdAssets200Ok
	Name string
}

func (s *CharacterService) updateCharacterAssetsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionAssets {
		panic("called with wrong section")
	}
	hasChanged, err := s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			assets, err := fetchFromESIWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdAssetsOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
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
			incomingIDs := set.New[int64]()
			for _, ca := range assets {
				incomingIDs.Add(ca.ItemId)
			}
			typeIDs := set.New[int32]()
			locationIDs := set.New[int64]()
			for _, ca := range assets {
				typeIDs.Add(ca.TypeId)
				if !incomingIDs.Contains(ca.LocationId) {
					locationIDs.Add(ca.LocationId) // location IDs that are not referencing other itemIDs are locations
				}
			}
			missingLocationIDs, err := s.st.MissingEveLocations(ctx, locationIDs.ToSlice())
			if err != nil {
				return err
			}
			for _, id := range missingLocationIDs {
				_, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, id)
				if err != nil {
					return err
				}
			}
			if err := s.EveUniverseService.AddMissingEveTypes(ctx, typeIDs.ToSlice()); err != nil {
				return err
			}
			currentIDs, err := s.st.ListCharacterAssetIDs(ctx, characterID)
			if err != nil {
				return err
			}
			for _, a := range assets {
				if currentIDs.Contains(a.ItemId) {
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
			}
			if ids := currentIDs.Difference(incomingIDs); ids.Size() > 0 {
				if err := s.st.DeleteCharacterAssets(ctx, characterID, ids.ToSlice()); err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return false, err
	}
	_, err = s.UpdateCharacterAssetTotalValue(ctx, arg.CharacterID)
	if err != nil {
		slog.Error("Failed to update asset total value", "characterID", arg.CharacterID, "err", err)
	}
	return hasChanged, err
}

func (s *CharacterService) fetchCharacterAssetNamesESI(ctx context.Context, characterID int32, ids []int64) (map[int64]string, error) {
	numResults := len(ids) / assetNamesMaxIDs
	if len(ids)%assetNamesMaxIDs > 0 {
		numResults++
	}
	results := make([][]esi.PostCharactersCharacterIdAssetsNames200Ok, numResults)
	g := new(errgroup.Group)
	for num, chunk := range Count(slices.Chunk(ids, assetNamesMaxIDs), 0) {
		g.Go(func() error {
			names, _, err := s.esiClient.ESI.AssetsApi.PostCharactersCharacterIdAssetsNames(ctx, characterID, chunk, nil)
			if err != nil {
				return err
			}
			results[num] = names
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		// We can live temporarily without asset names and will try again to fetch them next time
		// If some of the requests have succeeded we will use those names
		slog.Warn("Failed to fetch asset names", "characterID", characterID, "err", err)
	}
	m := make(map[int64]string)
	for _, names := range results {
		for _, n := range names {
			if n.Name != "None" {
				m[n.ItemId] = n.Name
			}
		}
	}
	return m, nil
}

func (s *CharacterService) CharacterAssetTotalValue(ctx context.Context, characterID int32) (optional.Optional[float64], error) {
	return s.st.GetCharacterAssetValue(ctx, characterID)
}

func (s *CharacterService) UpdateCharacterAssetTotalValue(ctx context.Context, characterID int32) (float64, error) {
	v, err := s.st.CalculateCharacterAssetTotalValue(ctx, characterID)
	if err != nil {
		return 0, err
	}
	if err := s.st.UpdateCharacterAssetValue(ctx, characterID, optional.New(v)); err != nil {
		return 0, err
	}
	return v, nil
}
