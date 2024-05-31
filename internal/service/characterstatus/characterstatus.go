// Package characterstatus caches for the update status of all characters.
package characterstatus

import (
	"context"
	"sync"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

const keyCharacters = "characterUpdateStatusCache-characters"

// CharacterStatus caches the current update status of all characters
// to improve performance of UI refresh tickers.
type CharacterStatus struct {
	cache *cache.Cache
	mu    sync.Mutex
}

type cacheKey struct {
	characterID int32
	section     model.CharacterSection
}

type cacheValue struct {
	ErrorMessage  string
	LastUpdatedAt time.Time
}

func New(cache *cache.Cache) *CharacterStatus {
	sc := &CharacterStatus{
		cache: cache,
	}
	return sc
}

func (sc *CharacterStatus) InitCache(r *storage.Storage) error {
	ctx := context.Background()
	ids, err := sc.updateCharacterIDs(ctx, r)
	if err != nil {
		return err
	}
	for _, characterID := range ids {
		oo, err := r.ListCharacterUpdateStatus(ctx, characterID)
		if err != nil {
			return err
		}
		for _, o := range oo {
			sc.SetStatus(characterID, o.Section, o.ErrorMessage, o.LastUpdatedAt)
		}
	}
	return nil
}

func (sc *CharacterStatus) GetStatus(characterID int32, section model.CharacterSection) (string, time.Time) {
	k := cacheKey{characterID: characterID, section: section}
	x, ok := sc.cache.Get(k)
	if !ok {
		return "", time.Time{}
	}
	v := x.(cacheValue)
	return v.ErrorMessage, v.LastUpdatedAt
}

func (sc *CharacterStatus) StatusSummary() (float32, bool) {
	ids := sc.GetCharacterIDs()
	total := len(model.CharacterSections) * len(ids)
	currentCount := 0
	for _, id := range ids {
		xx := sc.ListStatus(id)
		for _, x := range xx {
			if !x.IsOK() {
				return 0, false
			}
			if x.IsCurrent() {
				currentCount++
			}
		}
	}
	return float32(currentCount) / float32(total), true
}

func (sc *CharacterStatus) StatusCharacterSummary(characterID int32) (float32, bool) {
	total := len(model.CharacterSections)
	currentCount := 0
	xx := sc.ListStatus(characterID)
	for _, x := range xx {
		if !x.IsOK() {
			return 0, false
		}
		if x.IsCurrent() {
			currentCount++
		}
	}
	return float32(currentCount) / float32(total), true
}

func (sc *CharacterStatus) ListStatus(characterID int32) []model.CharacterStatus {
	list := make([]model.CharacterStatus, len(model.CharacterSections))
	for i, section := range model.CharacterSections {
		errorMessage, lastUpdatedAt := sc.GetStatus(characterID, section)
		list[i] = model.CharacterStatus{
			ErrorMessage:  errorMessage,
			LastUpdatedAt: lastUpdatedAt,
			Section:       section.Name(),
			Timeout:       section.Timeout(),
		}
	}
	return list
}

func (sc *CharacterStatus) SetStatus(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
	lastUpdatedAt time.Time,
) {
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, cache.NoTimeout)
}

func (sc *CharacterStatus) SetError(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	_, lastUpdatedAt := sc.GetStatus(characterID, section)
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, cache.NoTimeout)
}

func (sc *CharacterStatus) GetCharacterIDs() []int32 {
	x, ok := sc.cache.Get(keyCharacters)
	if !ok {
		return []int32{}
	}
	ids := x.([]int32)
	return ids
}

func (sc *CharacterStatus) SetCharacterIDs(ids []int32) {
	sc.cache.Set(keyCharacters, ids, cache.NoTimeout)
}

func (sc *CharacterStatus) UpdateCharacterIDs(ctx context.Context, r *storage.Storage) error {
	_, err := sc.updateCharacterIDs(ctx, r)
	return err
}

func (sc *CharacterStatus) updateCharacterIDs(ctx context.Context, r *storage.Storage) ([]int32, error) {
	ids, err := r.ListCharacterIDs(ctx)
	if err != nil {
		return nil, err
	}
	sc.cache.Set(keyCharacters, ids, cache.NoTimeout)
	return ids, nil
}
