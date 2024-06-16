// Package characterstatus is a service which provides cached access
// to the current update status of general and character sections.
package statuscache

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type StatusCacheStorage interface {
	ListCharacterSectionStatus(context.Context, int32) ([]*app.CharacterSectionStatus, error)
	ListGeneralSectionStatus(context.Context) ([]*app.GeneralSectionStatus, error)
	ListCharactersShort(context.Context) ([]*app.CharacterShort, error)
}

type Cache interface {
	Get(any) (any, bool)
	Set(any, any, time.Duration)
}

const keyCharacters = "characterUpdateStatusCache-characters"

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
func (sc *StatusCacheService) InitCache(st StatusCacheStorage) error {
	ctx := context.Background()
	cc, err := sc.updateCharacters(ctx, st)
	if err != nil {
		return err
	}
	for _, c := range cc {
		oo, err := st.ListCharacterSectionStatus(ctx, c.ID)
		if err != nil {
			return err
		}
		for _, o := range oo {
			sc.CharacterSectionSet(o)
		}
	}
	oo, err := st.ListGeneralSectionStatus(ctx)
	if err != nil {
		return err
	}
	for _, o := range oo {
		sc.GeneralSectionSet(o)
	}
	return nil
}

func (sc *StatusCacheService) CharacterSectionExists(characterID int32, section app.CharacterSection) bool {
	x, ok := sc.CharacterSectionGet(characterID, section)
	if !ok {
		return false
	}
	return !x.IsMissing()
}

func (sc *StatusCacheService) CharacterSectionGet(characterID int32, section app.CharacterSection) (SectionStatus, bool) {
	k := cacheKey{id: characterID, section: string(section)}
	x, ok := sc.cache.Get(k)
	if !ok {
		return SectionStatus{}, false
	}
	v := x.(cacheValue)
	o := SectionStatus{
		EntityID:     characterID,
		EntityName:   sc.characterName(characterID),
		SectionID:    string(section),
		SectionName:  section.DisplayName(),
		CompletedAt:  v.CompletedAt,
		ErrorMessage: v.ErrorMessage,
		StartedAt:    v.StartedAt,
		Timeout:      section.Timeout(),
	}
	return o, true
}

func (sc *StatusCacheService) CharacterSectionList(characterID int32) []SectionStatus {
	list := make([]SectionStatus, 0)
	for _, section := range app.CharacterSections {
		v, ok := sc.CharacterSectionGet(characterID, section)
		if !ok {
			continue
		}
		list = append(list, v)
	}
	return list
}

func (sc *StatusCacheService) CharacterSectionSummary(characterID int32) StatusSummary {
	total := len(app.CharacterSections)
	currentCount, missingCount, errorCount := sc.characterSectionSummary(characterID)
	s := StatusSummary{
		Current: currentCount,
		Errors:  errorCount,
		Missing: missingCount,
		Total:   total,
	}
	return s
}

func (sc *StatusCacheService) characterSectionSummary(characterID int32) (int, int, int) {
	currentCount := 0
	errorCount := 0
	missingCount := 0
	xx := sc.CharacterSectionList(characterID)
	for _, x := range xx {
		if !x.IsOK() {
			errorCount++
		} else if x.IsMissing() {
			missingCount++
		} else if x.IsCurrent() {
			currentCount++
		}
	}
	if diff := len(app.CharacterSections) - len(xx); diff > 0 {
		missingCount += diff
	}
	return currentCount, missingCount, errorCount
}

func (sc *StatusCacheService) CharacterSectionSet(o *app.CharacterSectionStatus) {
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

func (sc *StatusCacheService) GeneralSectionExists(section app.GeneralSection) bool {
	x, ok := sc.GeneralSectionGet(section)
	if !ok {
		return false
	}
	return !x.IsMissing()
}

func (sc *StatusCacheService) GeneralSectionGet(section app.GeneralSection) (SectionStatus, bool) {
	k := cacheKey{id: GeneralSectionEntityID, section: string(section)}
	x, ok := sc.cache.Get(k)
	if !ok {
		return SectionStatus{}, false
	}
	v := x.(cacheValue)
	o := SectionStatus{
		EntityID:     GeneralSectionEntityID,
		EntityName:   GeneralSectionEntityName,
		SectionID:    string(section),
		SectionName:  section.DisplayName(),
		CompletedAt:  v.CompletedAt,
		ErrorMessage: v.ErrorMessage,
		StartedAt:    v.StartedAt,
		Timeout:      section.Timeout(),
	}
	return o, true
}

func (sc *StatusCacheService) GeneralSectionList() []SectionStatus {
	list := make([]SectionStatus, 0)
	for _, section := range app.GeneralSections {
		v, ok := sc.GeneralSectionGet(section)
		if ok {
			list = append(list, v)
		}
	}
	return list
}

func (sc *StatusCacheService) GeneralSectionSet(o *app.GeneralSectionStatus) {
	if o == nil {
		return
	}
	k := cacheKey{
		id:      GeneralSectionEntityID,
		section: string(o.Section),
	}
	v := cacheValue{
		ErrorMessage: o.ErrorMessage,
		CompletedAt:  o.CompletedAt,
		StartedAt:    o.StartedAt,
	}
	sc.cache.Set(k, v, 0)
}

func (sc *StatusCacheService) GeneralSectionSummary() StatusSummary {
	total := len(app.GeneralSections)
	currentCount, missingCount, errorCount := sc.generalSectionSummary()
	s := StatusSummary{
		Current: currentCount,
		Errors:  errorCount,
		Missing: missingCount,
		Total:   total,
	}
	return s
}

func (sc *StatusCacheService) generalSectionSummary() (int, int, int) {
	currentCount := 0
	errorCount := 0
	missingCount := 0
	xx := sc.GeneralSectionList()
	for _, x := range xx {
		if !x.IsOK() {
			errorCount++
		} else if x.IsMissing() {
			missingCount++
		} else if x.IsCurrent() {
			currentCount++
		}
	}
	if diff := len(app.GeneralSections) - len(xx); diff > 0 {
		missingCount += diff
	}
	return currentCount, missingCount, errorCount
}

func (sc *StatusCacheService) SectionList(entityID int32) []SectionStatus {
	if entityID == GeneralSectionEntityID {
		return sc.GeneralSectionList()
	}
	return sc.CharacterSectionList(entityID)
}

// Summary returns the current summary status in percent of fresh sections
// and the number of missing and errors.
func (sc *StatusCacheService) Summary() StatusSummary {
	cc := sc.ListCharacters()
	currentCount := 0
	errorCount := 0
	missingCount := 0
	for _, character := range cc {
		c, m, e := sc.characterSectionSummary(character.ID)
		currentCount += c
		missingCount += m
		errorCount += e
	}
	c, m, e := sc.generalSectionSummary()
	currentCount += c
	missingCount += m
	errorCount += e
	total := len(app.CharacterSections)*len(cc) + len(app.GeneralSections)
	s := StatusSummary{
		Current: currentCount,
		Errors:  errorCount,
		Missing: missingCount,
		Total:   total,
	}
	return s
}

func (sc *StatusCacheService) UpdateCharacters(ctx context.Context, r StatusCacheStorage) error {
	_, err := sc.updateCharacters(ctx, r)
	return err
}

func (sc *StatusCacheService) updateCharacters(ctx context.Context, r StatusCacheStorage) ([]*app.CharacterShort, error) {
	cc, err := r.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	sc.setCharacters(cc)
	return cc, nil
}

func (sc *StatusCacheService) ListCharacters() []*app.CharacterShort {
	x, ok := sc.cache.Get(keyCharacters)
	if !ok {
		return nil
	}
	return x.([]*app.CharacterShort)
}

func (sc *StatusCacheService) setCharacters(cc []*app.CharacterShort) {
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
