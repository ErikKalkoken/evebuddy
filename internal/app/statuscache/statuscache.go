// Package characterstatus is a service which provides cached access
// to the current update status of general and character sections.
package statuscache

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

const keyCharacters = "characterUpdateStatusCache-characters"

// StatusCacheService provides cached access to the current update status
// of all characters to improve performance of UI refresh tickers.
type StatusCacheService struct {
	cache app.CacheService
}

// New creates and returns a new instance of a character status service.
// When nil is provided it will create and use it's own cache instance.
func New(cache app.CacheService) *StatusCacheService {
	sc := &StatusCacheService{cache: cache}
	return sc
}

// InitCache initializes the internal state from local storage.
// It should always be called once for a new instance to ensure the cache is current.
func (sc *StatusCacheService) InitCache(ctx context.Context, st app.StatusCacheStorage) error {
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

// CharacterSectionExists reports wether a character section exists.
func (sc *StatusCacheService) CharacterSectionExists(characterID int32, section app.CharacterSection) bool {
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

type cacheValue struct {
	CompletedAt  time.Time
	ErrorMessage string
	StartedAt    time.Time
}

func (sc *StatusCacheService) CharacterSectionGet(characterID int32, section app.CharacterSection) (app.SectionStatus, bool) {
	k := cacheKey{id: characterID, section: string(section)}
	x, ok := sc.cache.Get(k)
	if !ok {
		return app.SectionStatus{}, false
	}
	v := x.(cacheValue)
	o := app.SectionStatus{
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
	missing   int
	isRunning bool
}

// add ads the content of another statusSummary (Mutating).
func (ss *statusSummary) add(other statusSummary) {
	ss.current += other.current
	ss.errors += other.errors
	ss.missing += other.missing
	ss.isRunning = ss.isRunning || other.isRunning
}

func (sc *StatusCacheService) CharacterSectionSummary(characterID int32) app.StatusSummary {
	total := len(app.CharacterSections)
	ss := sc.calcCharacterSectionSummary(characterID)
	s := app.StatusSummary{
		Current:   ss.current,
		Errors:    ss.errors,
		IsRunning: ss.isRunning,
		Missing:   ss.missing,
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
		} else if o.IsMissing() {
			ss.missing++
		} else if o.IsCurrent() {
			ss.current++
		}
		if o.IsRunning() {
			ss.isRunning = true
		}
	}
	if diff := len(app.CharacterSections) - len(csl); diff > 0 {
		ss.missing += diff
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
	sc.cache.Set(k, v, 0)
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
	x, ok := sc.cache.Get(k)
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
	sc.cache.Set(k, v, 0)
}

func (sc *StatusCacheService) GeneralSectionSummary() app.StatusSummary {
	ss := sc.calcGeneralSectionSummary()
	s := app.StatusSummary{
		Current:   ss.current,
		Errors:    ss.errors,
		Missing:   ss.missing,
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
		} else if o.IsMissing() {
			ss.missing++
		} else if o.IsCurrent() {
			ss.current++
		}
		if o.IsRunning() {
			ss.isRunning = true
		}
	}
	if diff := len(app.GeneralSections) - len(gsl); diff > 0 {
		ss.missing += diff
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
		Missing:   ss.missing,
		IsRunning: ss.isRunning,
		Total:     len(app.CharacterSections)*len(cc) + len(app.GeneralSections),
	}
	return s
}

func (sc *StatusCacheService) UpdateCharacters(ctx context.Context, r app.StatusCacheStorage) error {
	_, err := sc.updateCharacters(ctx, r)
	return err
}

func (sc *StatusCacheService) updateCharacters(ctx context.Context, r app.StatusCacheStorage) ([]*app.CharacterShort, error) {
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
