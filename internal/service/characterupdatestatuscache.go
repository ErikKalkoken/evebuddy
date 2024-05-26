package service

import (
	"context"
	"sync"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

type characterUpdateStatusCache struct {
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

const keyCharacters = "characterUpdateStatusCache-characters"

func newCharacterUpdateStatusCache(cache *cache.Cache) *characterUpdateStatusCache {
	sc := &characterUpdateStatusCache{
		cache: cache,
	}
	return sc
}

// TODO: Combine queries for better performance

func (sc *characterUpdateStatusCache) initCache(r *storage.Storage) error {
	ctx := context.Background()
	ids, err := r.ListCharacterIDs(ctx)
	if err != nil {
		return err
	}
	sc.setCharacterIDs(ids)
	for _, characterID := range ids {
		oo, err := r.ListCharacterUpdateStatus(ctx, characterID)
		if err != nil {
			return err
		}
		for _, o := range oo {
			sc.setStatus(characterID, o.Section, o.ErrorMessage, o.LastUpdatedAt)
		}
	}
	return nil
}

func (sc *characterUpdateStatusCache) getStatus(characterID int32, section model.CharacterSection) (string, time.Time) {
	k := cacheKey{characterID: characterID, section: section}
	x, ok := sc.cache.Get(k)
	if !ok {
		return "", time.Time{}
	}
	v := x.(cacheValue)
	return v.ErrorMessage, v.LastUpdatedAt
}

func (sc *characterUpdateStatusCache) setStatus(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
	lastUpdatedAt time.Time,
) {
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, cache.NoTimeout)
}

func (sc *characterUpdateStatusCache) setStatusError(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	_, lastUpdatedAt := sc.getStatus(characterID, section)
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, cache.NoTimeout)
}

func (sc *characterUpdateStatusCache) getCharacterIDs() []int32 {
	x, ok := sc.cache.Get(keyCharacters)
	if !ok {
		return []int32{}
	}
	ids := x.([]int32)
	return ids
}

func (sc *characterUpdateStatusCache) setCharacterIDs(ids []int32) {
	sc.cache.Set(keyCharacters, ids, cache.NoTimeout)
}
