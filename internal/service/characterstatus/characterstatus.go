// Package characterstatuscache defines a mechanism for caching the update status of characters.
package characterstatus

import (
	"context"
	"sync"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

type Cache struct {
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

func New(cache *cache.Cache) *Cache {
	sc := &Cache{
		cache: cache,
	}
	return sc
}

// TODO: Combine queries for better performance

func (sc *Cache) InitCache(r *storage.Storage) error {
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
			sc.Set(characterID, o.Section, o.ErrorMessage, o.LastUpdatedAt)
		}
	}
	return nil
}

func (sc *Cache) Get(characterID int32, section model.CharacterSection) (string, time.Time) {
	k := cacheKey{characterID: characterID, section: section}
	x, ok := sc.cache.Get(k)
	if !ok {
		return "", time.Time{}
	}
	v := x.(cacheValue)
	return v.ErrorMessage, v.LastUpdatedAt
}

func (sc *Cache) Set(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
	lastUpdatedAt time.Time,
) {
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, cache.NoTimeout)
}

func (sc *Cache) SetError(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	_, lastUpdatedAt := sc.Get(characterID, section)
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, cache.NoTimeout)
}

func (sc *Cache) GetCharacterIDs() []int32 {
	x, ok := sc.cache.Get(keyCharacters)
	if !ok {
		return []int32{}
	}
	ids := x.([]int32)
	return ids
}

func (sc *Cache) SetCharacterIDs(ids []int32) {
	sc.cache.Set(keyCharacters, ids, cache.NoTimeout)
}

func (sc *Cache) UpdateCharacterIDs(ctx context.Context, r *storage.Storage) error {
	_, err := sc.updateCharacterIDs(ctx, r)
	return err
}

func (sc *Cache) updateCharacterIDs(ctx context.Context, r *storage.Storage) ([]int32, error) {
	ids, err := r.ListCharacterIDs(ctx)
	if err != nil {
		return nil, err
	}
	sc.cache.Set(keyCharacters, ids, cache.NoTimeout)
	return ids, nil
}
