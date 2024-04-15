package service

import (
	"context"
	"errors"
	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/repository"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
)

type EveEntityCategory int

// Supported categories of EveEntity
const (
	EveEntityAlliance EveEntityCategory = iota + 1
	EveEntityCharacter
	EveEntityCorporation
	EveEntityFaction
	EveEntityMailList
)

type EveEntity struct {
	Category EveEntityCategory
	ID       int32
	Name     string
}

func (e *EveEntity) IconURL(size int) (fyne.URI, error) {
	switch e.Category {
	case EveEntityAlliance:
		return images.AllianceLogoURL(e.ID, size)
	case EveEntityCharacter:
		return images.CharacterPortraitURL(e.ID, size)
	case EveEntityCorporation:
		return images.CorporationLogoURL(e.ID, size)
	case EveEntityFaction:
		return images.FactionLogoURL(e.ID, size)
	}
	return nil, errors.New("can not match category")
}

func eveEntityFromDBModel(e repository.EveEntity) EveEntity {
	if e.ID == 0 {
		return EveEntity{}
	}
	category := eveEntityCategoryFromDBModel(e.Category)
	return EveEntity{
		Category: category,
		ID:       int32(e.ID),
		Name:     e.Name,
	}
}

func eveEntityCategoryFromDBModel(c string) EveEntityCategory {
	categoryMap := map[string]EveEntityCategory{
		repository.EveEntityAlliance:    EveEntityAlliance,
		repository.EveEntityCharacter:   EveEntityCharacter,
		repository.EveEntityCorporation: EveEntityCorporation,
		repository.EveEntityFaction:     EveEntityFaction,
		repository.EveEntityMailList:    EveEntityMailList,
	}
	c2, ok := categoryMap[c]
	if !ok {
		panic(fmt.Sprintf("Can not map unknown category: %s", c))
	}
	return c2
}

func eveEntityDBModelCategoryFromCategory(c EveEntityCategory) string {
	categoryMap := map[EveEntityCategory]string{
		EveEntityAlliance:    repository.EveEntityAlliance,
		EveEntityCharacter:   repository.EveEntityCharacter,
		EveEntityCorporation: repository.EveEntityCorporation,
		EveEntityFaction:     repository.EveEntityFaction,
		EveEntityMailList:    repository.EveEntityMailList,
	}
	c2, ok := categoryMap[c]
	if !ok {
		panic(fmt.Sprintf("Can not map unknown category: %v", c))
	}
	return c2
}

func eveEntityESCategoryFromESICategory(c string) string {
	categoryMap := map[string]string{
		"alliance":     repository.EveEntityAlliance,
		"character":    repository.EveEntityCharacter,
		"corporation":  repository.EveEntityCorporation,
		"faction":      repository.EveEntityFaction,
		"mailing:list": repository.EveEntityMailList,
	}
	c2, ok := categoryMap[c]
	if !ok {
		panic(fmt.Sprintf("Can not map unknown category: %v", c))
	}
	return c2
}

// AddEveEntitiesFromESISearch runs a search on ESI and adds the results as new EveEntity objects to the database.
func (s *Service) AddEveEntitiesFromESISearch(characterID int32, search string) ([]int32, error) {
	token, err := s.GetValidToken(characterID)
	if err != nil {
		return nil, err
	}
	categories := []string{
		"corporation",
		"character",
		"alliance",
	}
	r, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(token.NewContext(), categories, characterID, search, nil)
	if err != nil {
		return nil, err
	}
	ids := slices.Concat(r.Alliance, r.Character, r.Corporation)
	missingIDs, err := s.addMissingEveEntities(ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs, nil
}

// addMissingEveEntities adds EveEntities from ESI for IDs missing in the database.
func (s *Service) addMissingEveEntities(ids []int32) ([]int32, error) {
	c, err := s.queries.ListEveEntityIDs(context.Background())
	if err != nil {
		return nil, err
	}
	c2 := make([]int32, len(c))
	for i, id := range c {
		c2[i] = int32(id)
	}
	current := set.NewFromSlice(c2)
	incoming := set.NewFromSlice(ids)
	missing := incoming.Difference(current)

	if missing.Size() == 0 {
		return nil, nil
	}

	entities, _, err := s.esiClient.ESI.UniverseApi.PostUniverseNames(context.Background(), missing.ToSlice(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve IDs: %v %v", err, ids)
	}
	for _, entity := range entities {
		arg := repository.CreateEveEntityParams{
			ID:       int64(entity.Id),
			Category: eveEntityESCategoryFromESICategory(entity.Category),
			Name:     entity.Name,
		}
		err := s.queries.CreateEveEntity(context.Background(), arg)
		if err != nil {
			return nil, err
		}
	}
	slog.Debug("Added missing eve entities", "count", len(entities))
	return missing.ToSlice(), nil
}

func (s *Service) SearchEveEntitiesByName(partial string) ([]EveEntity, error) {
	ee, err := s.queries.ListEveEntitiesByPartialName(context.Background(), partial)
	if err != nil {
		return nil, err
	}
	ee2 := make([]EveEntity, len(ee))
	for i, e := range ee {
		ee2[i] = eveEntityFromDBModel(e)
	}
	return ee2, nil
}
