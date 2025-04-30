// Package statuscacheservice is a service which provides cached access
// to the current update status of general and character sections.
package statuscacheservice

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

const keyCharacters = "characterUpdateStatusCache-characters"

// StatusCacheService provides cached access to the current update status
// of all characters to improve performance of UI refresh tickers.
type StatusCacheService struct {
	cache *memcache.Cache
	st    *storage.Storage
}

// New creates and returns a new instance of a character status service.
// When nil is provided it will create and use it's own cache instance.
func New(cache *memcache.Cache, st *storage.Storage) *StatusCacheService {
	sc := &StatusCacheService{cache: cache, st: st}
	return sc
}

// InitCache initializes the internal state from local storage.
// It should always be called once for a new instance to ensure the cache is current.
func (sc *StatusCacheService) InitCache(ctx context.Context) error {
	cc, err := sc.updateCharacters(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		oo, err := sc.st.ListCharacterSectionStatus(ctx, c.ID)
		if err != nil {
			return err
		}
		for _, o := range oo {
			sc.CharacterSectionSet(o)
		}
	}
	oo, err := sc.st.ListGeneralSectionStatus(ctx)
	if err != nil {
		return err
	}
	for _, o := range oo {
		sc.GeneralSectionSet(o)
	}
	return nil
}

// CharacterSectionExists reports whether a character section exists.
func (sc *StatusCacheService) CharacterSectionExists(characterID int32, section app.CharacterSection) bool {
	if characterID == 0 {
		return false
	}
	x, ok := sc.CharacterSectionGet(characterID, section)
	if !ok {
		return false
	}
	return !x.IsMissing()
}

type cacheKey struct {
	id      int32
	section string
}

func (ck cacheKey) String() string {
	return fmt.Sprintf("%d-%s", ck.id, ck.section)
}

type cacheValue struct {
	CompletedAt  time.Time
	ErrorMessage string
	StartedAt    time.Time
}

func (sc *StatusCacheService) CharacterSectionGet(characterID int32, section app.CharacterSection) (app.SectionStatus, bool) {
	k := cacheKey{id: characterID, section: string(section)}
	x, ok := sc.cache.Get(k.String())
	if !ok {
		return app.SectionStatus{}, false
	}
	v := x.(cacheValue)
	o := app.SectionStatus{
		EntityID:     characterID,
		EntityName:   sc.CharacterName(characterID),
		SectionID:    string(section),
		SectionName:  section.DisplayName(),
		CompletedAt:  v.CompletedAt,
		ErrorMessage: v.ErrorMessage,
		StartedAt:    v.StartedAt,
		Timeout:      section.Timeout(),
	}
	return o, true
}

func (sc *StatusCacheService) CharacterSectionList(characterID int32) []app.SectionStatus {
	list := make([]app.SectionStatus, 0)
	for _, section := range app.CharacterSections {
		v, ok := sc.CharacterSectionGet(characterID, section)
		if !ok {
			continue
		}
		list = append(list, v)
	}
	return list
}

type statusSummary struct {
	current   int
	errors    int
	isRunning bool
}

// add ads the content of another statusSummary (Mutating).
func (ss *statusSummary) add(other statusSummary) {
	ss.current += other.current
	ss.errors += other.errors
	ss.isRunning = ss.isRunning || other.isRunning
}

func (sc *StatusCacheService) CharacterSectionSummary(characterID int32) app.StatusSummary {
	total := len(app.CharacterSections)
	ss := sc.calcCharacterSectionSummary(characterID)
	s := app.StatusSummary{
		Current:   ss.current,
		Errors:    ss.errors,
		IsRunning: ss.isRunning,
		Total:     total,
	}
	return s
}

func (sc *StatusCacheService) calcCharacterSectionSummary(characterID int32) statusSummary {
	var ss statusSummary
	csl := sc.CharacterSectionList(characterID)
	for _, o := range csl {
		if !o.IsOK() {
			ss.errors++
		} else if o.IsCurrent() {
			ss.current++
		}
		if o.IsRunning() {
			ss.isRunning = true
		}
	}
	return ss
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
	sc.cache.Set(k.String(), v, 0)
}

// Return the name of a character by ID or an empty string if not found.
func (sc *StatusCacheService) CharacterName(characterID int32) string {
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

func (sc *StatusCacheService) GeneralSectionExists(section app.GeneralSection) bool {
	x, ok := sc.GeneralSectionGet(section)
	if !ok {
		return false
	}
	return !x.IsMissing()
}

func (sc *StatusCacheService) GeneralSectionGet(section app.GeneralSection) (app.SectionStatus, bool) {
	k := cacheKey{id: app.GeneralSectionEntityID, section: string(section)}
	x, ok := sc.cache.Get(k.String())
	if !ok {
		return app.SectionStatus{}, false
	}
	v := x.(cacheValue)
	o := app.SectionStatus{
		EntityID:     app.GeneralSectionEntityID,
		EntityName:   app.GeneralSectionEntityName,
		SectionID:    string(section),
		SectionName:  section.DisplayName(),
		CompletedAt:  v.CompletedAt,
		ErrorMessage: v.ErrorMessage,
		StartedAt:    v.StartedAt,
		Timeout:      section.Timeout(),
	}
	return o, true
}

func (sc *StatusCacheService) GeneralSectionList() []app.SectionStatus {
	list := make([]app.SectionStatus, 0)
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
		id:      app.GeneralSectionEntityID,
		section: string(o.Section),
	}
	v := cacheValue{
		ErrorMessage: o.ErrorMessage,
		CompletedAt:  o.CompletedAt,
		StartedAt:    o.StartedAt,
	}
	sc.cache.Set(k.String(), v, 0)
}

func (sc *StatusCacheService) GeneralSectionSummary() app.StatusSummary {
	ss := sc.calcGeneralSectionSummary()
	s := app.StatusSummary{
		Current:   ss.current,
		Errors:    ss.errors,
		IsRunning: ss.isRunning,
		Total:     len(app.GeneralSections),
	}
	return s
}

func (sc *StatusCacheService) calcGeneralSectionSummary() statusSummary {
	var ss statusSummary
	gsl := sc.GeneralSectionList()
	for _, o := range gsl {
		if !o.IsOK() {
			ss.errors++
		} else if o.IsCurrent() {
			ss.current++
		}
		if o.IsRunning() {
			ss.isRunning = true
		}
	}
	return ss
}

func (sc *StatusCacheService) SectionList(entityID int32) []app.SectionStatus {
	if entityID == app.GeneralSectionEntityID {
		return sc.GeneralSectionList()
	}
	return sc.CharacterSectionList(entityID)
}

// Summary returns the current summary status in percent of fresh sections
// and the number of missing and errors.
func (sc *StatusCacheService) Summary() app.StatusSummary {
	var ss statusSummary
	cc := sc.ListCharacters()
	for _, character := range cc {
		ss.add(sc.calcCharacterSectionSummary(character.ID))
	}
	ss.add(sc.calcGeneralSectionSummary())
	s := app.StatusSummary{
		Current:   ss.current,
		Errors:    ss.errors,
		IsRunning: ss.isRunning,
		Total:     len(app.CharacterSections)*len(cc) + len(app.GeneralSections),
	}
	return s
}

func (sc *StatusCacheService) UpdateCharacters(ctx context.Context) error {
	_, err := sc.updateCharacters(ctx)
	return err
}

func (sc *StatusCacheService) updateCharacters(ctx context.Context) ([]*app.CharacterShort, error) {
	cc, err := sc.st.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	sc.setCharacters(cc)
	return cc, nil
}

// ListCharacters returns the list of the users characters in alphabetical order.
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
