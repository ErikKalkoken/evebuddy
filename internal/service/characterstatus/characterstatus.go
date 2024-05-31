// Package characterstatus provides information about the update status of all characters.
package characterstatus

import (
	"context"
	"sync"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type Storage interface {
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

// CharacterStatus caches the current update status of all characters
// to improve performance of UI refresh tickers.
type CharacterStatus struct {
	cache Cache
	mu    sync.Mutex
}

func New(cache Cache) *CharacterStatus {
	sc := &CharacterStatus{cache: cache}
	return sc
}

func (sc *CharacterStatus) InitCache(r Storage) error {
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

func (sc *CharacterStatus) GetStatus(characterID int32, section model.CharacterSection) (string, time.Time) {
	k := cacheKey{characterID: characterID, section: section}
	x, ok := sc.cache.Get(k)
	if !ok {
		return "", time.Time{}
	}
	v := x.(cacheValue)
	return v.ErrorMessage, v.LastUpdatedAt
}

func (sc *CharacterStatus) Summary() (float32, bool) {
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

func (sc *CharacterStatus) CharacterSummary(characterID int32) (float32, bool) {
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
	sc.cache.Set(k, v, 0)
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
	sc.cache.Set(k, v, 0)
}

func (sc *CharacterStatus) UpdateCharacters(ctx context.Context, r Storage) error {
	_, err := sc.updateCharacters(ctx, r)
	return err
}

func (sc *CharacterStatus) updateCharacters(ctx context.Context, r Storage) ([]*model.CharacterShort, error) {
	cc, err := r.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	sc.setCharacters(cc)
	return cc, nil
}

func (sc *CharacterStatus) ListCharacters() []*model.CharacterShort {
	x, ok := sc.cache.Get(keyCharacters)
	if !ok {
		return nil
	}
	return x.([]*model.CharacterShort)
}

func (sc *CharacterStatus) setCharacters(cc []*model.CharacterShort) {
	sc.cache.Set(keyCharacters, cc, 0)
}
