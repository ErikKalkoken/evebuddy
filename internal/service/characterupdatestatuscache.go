package service

import (
	"context"
	"sync"
	"time"

	icache "github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

type characterUpdateStatusCache struct {
	cache icache.Cache
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

func newCharacterUpdateStatusCache() *characterUpdateStatusCache {
	sc := &characterUpdateStatusCache{
		cache: *icache.New(),
	}
	return sc
}

// TODO: Combine queries for better performance

func (sc *characterUpdateStatusCache) initCache(r *storage.Storage) error {
	ctx := context.Background()
	cc, err := r.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		oo, err := r.ListCharacterUpdateStatus(ctx, c.ID)
		if err != nil {
			return err
		}
		for _, o := range oo {
			sc.set(c.ID, o.Section, o.ErrorMessage, o.LastUpdatedAt)
		}
	}
	return nil
}

func (sc *characterUpdateStatusCache) get(characterID int32, section model.CharacterSection) (string, time.Time) {
	k := cacheKey{characterID: characterID, section: section}
	x, ok := sc.cache.Get(k)
	if !ok {
		return "", time.Time{}
	}
	v := x.(cacheValue)
	return v.ErrorMessage, v.LastUpdatedAt
}

func (sc *characterUpdateStatusCache) set(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
	lastUpdatedAt time.Time,
) {
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, icache.NoTimeout)
}

func (sc *characterUpdateStatusCache) setError(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	_, lastUpdatedAt := sc.get(characterID, section)
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, icache.NoTimeout)
}
