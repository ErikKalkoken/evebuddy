// Package characterstatus provides information about the update status of all characters.
package characterstatus

import (
	"context"
	"sync"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type CharacterStatusStorage interface {
	ListCharacterUpdateStatus(context.Context, int32) ([]*model.CharacterUpdateStatus, error)
	ListCharactersShort(context.Context) ([]*model.CharacterShort, error)
}

type Cache interface {
	Get(any) (any, bool)
	Set(any, any, time.Duration)
}

const keyCharacters = "characterUpdateStatusCache-characters"

type cacheKey struct {
	characterID int32
	section     model.CharacterSection
}

type cacheValue struct {
	ErrorMessage  string
	LastUpdatedAt time.Time
}

// CharacterStatusCache caches the current update status of all characters
// to improve performance of UI refresh tickers.
type CharacterStatusCache struct {
	cache Cache
	mu    sync.Mutex
}

func New(cache Cache) *CharacterStatusCache {
	sc := &CharacterStatusCache{cache: cache}
	return sc
}

func (sc *CharacterStatusCache) InitCache(r CharacterStatusStorage) error {
	ctx := context.Background()
	cc, err := sc.updateCharacters(ctx, r)
	if err != nil {
		return err
	}
	for _, c := range cc {
		oo, err := r.ListCharacterUpdateStatus(ctx, c.ID)
		if err != nil {
			return err
		}
		for _, o := range oo {
			sc.SetStatus(c.ID, o.Section, o.ErrorMessage, o.LastUpdatedAt)
		}
	}
	return nil
}

func (sc *CharacterStatusCache) GetStatus(characterID int32, section model.CharacterSection) (string, time.Time) {
	k := cacheKey{characterID: characterID, section: section}
	x, ok := sc.cache.Get(k)
	if !ok {
		return "", time.Time{}
	}
	v := x.(cacheValue)
	return v.ErrorMessage, v.LastUpdatedAt
}

func (sc *CharacterStatusCache) Summary() (float32, bool) {
	cc := sc.ListCharacters()
	total := len(model.CharacterSections) * len(cc)
	currentCount := 0
	for _, c := range cc {
		xx := sc.ListStatus(c.ID)
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

func (sc *CharacterStatusCache) CharacterSummary(characterID int32) (float32, bool) {
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

func (sc *CharacterStatusCache) ListStatus(characterID int32) []model.CharacterStatus {
	characterName := sc.characterName(characterID)
	list := make([]model.CharacterStatus, len(model.CharacterSections))
	for i, section := range model.CharacterSections {
		errorMessage, lastUpdatedAt := sc.GetStatus(characterID, section)
		list[i] = model.CharacterStatus{
			CharacterID:   characterID,
			CharacterName: characterName,
			ErrorMessage:  errorMessage,
			LastUpdatedAt: lastUpdatedAt,
			Section:       section,
		}
	}
	return list
}

func (sc *CharacterStatusCache) SetStatus(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
	lastUpdatedAt time.Time,
) {
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, 0)
}

func (sc *CharacterStatusCache) SetError(
	characterID int32,
	section model.CharacterSection,
	errorMessage string,
) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	_, lastUpdatedAt := sc.GetStatus(characterID, section)
	k := cacheKey{characterID: characterID, section: section}
	v := cacheValue{ErrorMessage: errorMessage, LastUpdatedAt: lastUpdatedAt}
	sc.cache.Set(k, v, 0)
}

func (sc *CharacterStatusCache) UpdateCharacters(ctx context.Context, r CharacterStatusStorage) error {
	_, err := sc.updateCharacters(ctx, r)
	return err
}

func (sc *CharacterStatusCache) updateCharacters(ctx context.Context, r CharacterStatusStorage) ([]*model.CharacterShort, error) {
	cc, err := r.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	sc.setCharacters(cc)
	return cc, nil
}

func (sc *CharacterStatusCache) ListCharacters() []*model.CharacterShort {
	x, ok := sc.cache.Get(keyCharacters)
	if !ok {
		return nil
	}
	return x.([]*model.CharacterShort)
}

func (sc *CharacterStatusCache) setCharacters(cc []*model.CharacterShort) {
	sc.cache.Set(keyCharacters, cc, 0)
}

func (sc *CharacterStatusCache) characterName(characterID int32) string {
	cc := sc.ListCharacters()
	if len(cc) == 0 {
		return ""
	}
	for _, c := range cc {
		if c.ID == characterID {
			return c.Name
		}
	}
	return ""
}
