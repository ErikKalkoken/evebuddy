// Package characterstatus is a service which provides cached access
// to the current update status of general and character sections.
package statuscache

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type StatusCacheStorage interface {
	ListCharacterSectionStatus(context.Context, int32) ([]*model.CharacterSectionStatus, error)
	ListCharactersShort(context.Context) ([]*model.CharacterShort, error)
}

type Cache interface {
	Get(any) (any, bool)
	Set(any, any, time.Duration)
}

const (
	keyCharacters = "characterUpdateStatusCache-characters"
	eveUniverseID = 0
)

type cacheKey struct {
	id      int32
	section string
}

type cacheValue struct {
	CompletedAt  time.Time
	ErrorMessage string
	StartedAt    time.Time
}

// StatusCacheService provides cached access to the current update status
// of all characters to improve performance of UI refresh tickers.
type StatusCacheService struct {
	cache Cache
}

// New creates and returns a new instance of a character status service.
// When nil is provided it will create and use it's own cache instance.
func New(cache Cache) *StatusCacheService {
	sc := &StatusCacheService{cache: cache}
	return sc
}

// InitCache initializes the internal state from local storage.
// It should always be called once for a new instance to ensure the cache is current.
func (sc *StatusCacheService) InitCache(r StatusCacheStorage) error {
	ctx := context.Background()
	cc, err := sc.updateCharacters(ctx, r)
	if err != nil {
		return err
	}
	for _, c := range cc {
		oo, err := r.ListCharacterSectionStatus(ctx, c.ID)
		if err != nil {
			return err
		}
		for _, o := range oo {
			sc.CharacterSectionSet(o)
		}
	}
	return nil
}

func (sc *StatusCacheService) CharacterSectionGet(characterID int32, section model.CharacterSection) *model.CharacterSectionStatus {
	k := cacheKey{id: characterID, section: string(section)}
	x, ok := sc.cache.Get(k)
	if !ok {
		return nil
	}
	v := x.(cacheValue)
	return &model.CharacterSectionStatus{
		CharacterID:   characterID,
		CharacterName: sc.characterName(characterID),
		Section:       section,
		CompletedAt:   v.CompletedAt,
		ErrorMessage:  v.ErrorMessage,
		StartedAt:     v.StartedAt,
	}
}

func (sc *StatusCacheService) CharacterSectionSet(o *model.CharacterSectionStatus) {
	if o == nil {
		return
	}
	k := cacheKey{
		id:      o.CharacterID,
		section: string(o.Section),
	}
	v := cacheValue{
		ErrorMessage: o.ErrorMessage,
		CompletedAt:  o.CompletedAt,
		StartedAt:    o.StartedAt,
	}
	sc.cache.Set(k, v, 0)
}

func (sc *StatusCacheService) GeneralSectionSet(o *model.GeneralSectionStatus) {
	if o == nil {
		return
	}
	k := cacheKey{
		id:      eveUniverseID,
		section: string(o.Section),
	}
	v := cacheValue{
		ErrorMessage: o.ErrorMessage,
		CompletedAt:  o.CompletedAt,
		StartedAt:    o.StartedAt,
	}
	sc.cache.Set(k, v, 0)
}

func (sc *StatusCacheService) GeneralSectionGet(section model.GeneralSection) *model.GeneralSectionStatus {
	k := cacheKey{id: eveUniverseID, section: string(section)}
	x, ok := sc.cache.Get(k)
	if !ok {
		return nil
	}
	v := x.(cacheValue)
	return &model.GeneralSectionStatus{
		Section:      section,
		CompletedAt:  v.CompletedAt,
		ErrorMessage: v.ErrorMessage,
		StartedAt:    v.StartedAt,
	}
}

func (sc *StatusCacheService) Summary() (float32, int) {
	cc := sc.ListCharacters()
	sectionsTotal := len(model.CharacterSections) * len(cc)
	sectionsCurrent := 0
	errorCount := 0
	for _, c := range cc {
		xx := sc.CharacterSectionList(c.ID)
		for _, x := range xx {
			if !x.IsOK() {
				errorCount++
				continue
			}
			if x.IsCurrent() {
				sectionsCurrent++
			}
		}
	}
	return float32(sectionsCurrent) / float32(sectionsTotal), errorCount
}

func (sc *StatusCacheService) CharacterSectionSummary(characterID int32) (float32, bool) {
	total := len(model.CharacterSections)
	currentCount := 0
	xx := sc.CharacterSectionList(characterID)
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

func (sc *StatusCacheService) CharacterSectionList(characterID int32) []*model.CharacterSectionStatus {
	list := make([]*model.CharacterSectionStatus, len(model.CharacterSections))
	for i, section := range model.CharacterSections {
		v := sc.CharacterSectionGet(characterID, section)
		list[i] = v
	}
	return list
}

func (sc *StatusCacheService) UpdateCharacters(ctx context.Context, r StatusCacheStorage) error {
	_, err := sc.updateCharacters(ctx, r)
	return err
}

func (sc *StatusCacheService) updateCharacters(ctx context.Context, r StatusCacheStorage) ([]*model.CharacterShort, error) {
	cc, err := r.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	sc.setCharacters(cc)
	return cc, nil
}

func (sc *StatusCacheService) ListCharacters() []*model.CharacterShort {
	x, ok := sc.cache.Get(keyCharacters)
	if !ok {
		return nil
	}
	return x.([]*model.CharacterShort)
}

func (sc *StatusCacheService) setCharacters(cc []*model.CharacterShort) {
	sc.cache.Set(keyCharacters, cc, 0)
}

func (sc *StatusCacheService) characterName(characterID int32) string {
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
