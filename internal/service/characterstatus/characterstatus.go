// Package characterstatus contains the character status service.
package characterstatus

import (
	"context"
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
	CompletedAt  time.Time
	ErrorMessage string
	StartedAt    time.Time
	UpdatedAt    time.Time
}

// CharacterStatusService provides cached access to the current update status
// of all characters to improve performance of UI refresh tickers.
type CharacterStatusService struct {
	cache Cache
}

// New creates and returns a new instance of a character status service.
// When nil is provided it will create and use it's own cache instance.
func New(cache Cache) *CharacterStatusService {
	sc := &CharacterStatusService{cache: cache}
	return sc
}

// InitCache initializes the internal state from local storage.
// It should always be called once for a new instance to ensure the cache is current.
func (sc *CharacterStatusService) InitCache(r CharacterStatusStorage) error {
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
			sc.Set(o)
		}
	}
	return nil
}

func (sc *CharacterStatusService) Get(characterID int32, section model.CharacterSection) model.CharacterStatus {
	k := cacheKey{characterID: characterID, section: section}
	x, ok := sc.cache.Get(k)
	if !ok {
		return model.CharacterStatus{}
	}
	v := x.(cacheValue)
	return model.CharacterStatus{
		CharacterID:   characterID,
		CharacterName: sc.characterName(characterID),
		Section:       section,
		CompletedAt:   v.CompletedAt,
		ErrorMessage:  v.ErrorMessage,
		StartedAt:     v.StartedAt,
		UpdateAt:      v.UpdatedAt,
	}
}

func (sc *CharacterStatusService) Summary() (float32, int) {
	cc := sc.ListCharacters()
	sectionsTotal := len(model.CharacterSections) * len(cc)
	sectionsCurrent := 0
	errorCount := 0
	for _, c := range cc {
		xx := sc.ListStatus(c.ID)
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

func (sc *CharacterStatusService) CharacterSummary(characterID int32) (float32, bool) {
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

func (sc *CharacterStatusService) ListStatus(characterID int32) []model.CharacterStatus {
	list := make([]model.CharacterStatus, len(model.CharacterSections))
	for i, section := range model.CharacterSections {
		v := sc.Get(characterID, section)
		list[i] = v
	}
	return list
}

func (sc *CharacterStatusService) Set(o *model.CharacterUpdateStatus) {
	if o == nil {
		return
	}
	k := cacheKey{
		characterID: o.CharacterID,
		section:     o.Section,
	}
	v := cacheValue{
		ErrorMessage: o.ErrorMessage,
		CompletedAt:  o.CompletedAt,
		StartedAt:    o.StartedAt,
		UpdatedAt:    o.UpdatedAt,
	}
	sc.cache.Set(k, v, 0)
}

func (sc *CharacterStatusService) UpdateCharacters(ctx context.Context, r CharacterStatusStorage) error {
	_, err := sc.updateCharacters(ctx, r)
	return err
}

func (sc *CharacterStatusService) updateCharacters(ctx context.Context, r CharacterStatusStorage) ([]*model.CharacterShort, error) {
	cc, err := r.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	sc.setCharacters(cc)
	return cc, nil
}

func (sc *CharacterStatusService) ListCharacters() []*model.CharacterShort {
	x, ok := sc.cache.Get(keyCharacters)
	if !ok {
		return nil
	}
	return x.([]*model.CharacterShort)
}

func (sc *CharacterStatusService) setCharacters(cc []*model.CharacterShort) {
	sc.cache.Set(keyCharacters, cc, 0)
}

func (sc *CharacterStatusService) characterName(characterID int32) string {
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
