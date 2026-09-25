package app

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/maniartech/signals"

	"github.com/ErikKalkoken/go-set"
)

type CharacterSectionUpdated struct {
	CharacterID  int64
	Section      CharacterSection
	NeedsRefresh bool
}

type CorporationSectionUpdated struct {
	CorporationID int64
	Section       CorporationSection
	NeedsRefresh  bool
}

type EveUniverseSectionUpdated struct {
	Changed      set.Set[int64]
	Section      EveUniverseSection
	NeedsRefresh bool
}

// Signals represents the app's event signals.
// It is safe for concurrent use.
type Signals struct {
	// The app is initialized
	AppInit signals.Signal[struct{}]

	// The app is shutting down. Listeners should cancel and wait for any
	// background work they own before returning.
	AppShutdown signals.Signal[struct{}]

	// A character was added.
	CharacterAdded signals.Signal[*Character]

	// A Character field has changed (e.g. number of unread messages).
	CharacterChanged signals.Signal[int64]

	// A character was removed.
	CharacterRemoved signals.Signal[*EntityShort]

	// A character section has changed after an update from ESI.
	CharacterSectionChanged signals.Signal[CharacterSectionUpdated]

	// A character section has been updated from ESI.
	CharacterSectionUpdated signals.Signal[CharacterSectionUpdated]

	// A corporation has been added or removed.
	CorporationsChanged signals.Signal[struct{}]

	// A corporation section has changed after an update from ESI.
	CorporationSectionChanged signals.Signal[CorporationSectionUpdated]

	// A corporation section has been updated from ESI.
	CorporationSectionUpdated signals.Signal[CorporationSectionUpdated]

	// The current character was exchanged with another character or reset.
	CurrentCharacterExchanged signals.Signal[*Character]

	// The current corporation was exchanged with another corporation or reset.
	CurrentCorporationExchanged signals.Signal[*Corporation]

	// Application data has has been updated.
	DataUpdated signals.Signal[string]

	// An EveUniverse section has changed after an update from ESI.
	EveUniverseSectionChanged signals.Signal[EveUniverseSectionUpdated]

	// An EveUniverse section has been updated after an update from ESI.
	EveUniverseSectionUpdated signals.Signal[EveUniverseSectionUpdated]

	// Ticker for dynamic UI refresh has expired.
	RefreshTickerExpired signals.Signal[struct{}]

	// A tag as been added, removed or renamed.
	TagsChanged signals.Signal[struct{}]

	// A section update has started.
	UpdateStarted signals.Signal[string]

	// A section update has stopped.
	UpdateStopped signals.Signal[string]

	keyID        atomic.Uint64
	shuttingDown atomic.Bool
}

func NewSignals() *Signals {
	s := &Signals{
		AppShutdown: signals.New[struct{}](),
	}
	g := &s.shuttingDown
	s.AppInit = newGuardedSignal[struct{}](g)
	s.CharacterAdded = newGuardedSignal[*Character](g)
	s.CharacterChanged = newGuardedSignal[int64](g)
	s.CharacterRemoved = newGuardedSignal[*EntityShort](g)
	s.CharacterSectionChanged = newGuardedSignal[CharacterSectionUpdated](g)
	s.CharacterSectionUpdated = newGuardedSignal[CharacterSectionUpdated](g)
	s.CorporationsChanged = newGuardedSignal[struct{}](g)
	s.CorporationSectionChanged = newGuardedSignal[CorporationSectionUpdated](g)
	s.CorporationSectionUpdated = newGuardedSignal[CorporationSectionUpdated](g)
	s.CurrentCharacterExchanged = newGuardedSignal[*Character](g)
	s.CurrentCorporationExchanged = newGuardedSignal[*Corporation](g)
	s.DataUpdated = newGuardedSignal[string](g)
	s.EveUniverseSectionChanged = newGuardedSignal[EveUniverseSectionUpdated](g)
	s.EveUniverseSectionUpdated = newGuardedSignal[EveUniverseSectionUpdated](g)
	s.RefreshTickerExpired = newGuardedSignal[struct{}](g)
	s.TagsChanged = newGuardedSignal[struct{}](g)
	s.UpdateStarted = newGuardedSignal[string](g)
	s.UpdateStopped = newGuardedSignal[string](g)
	return s
}

// BeginShutdown makes Emit a no-op on every signal except AppShutdown. Needed
// because fyne.Do stops serializing onto the main thread once Fyne's quit sequence
// starts draining its dispatch queue.
func (s *Signals) BeginShutdown() {
	s.shuttingDown.Store(true)
}

func (s *Signals) IsShuttingDown() bool {
	return s.shuttingDown.Load()
}

// guardedSignal suppresses Emit once shuttingDown is set.
//
// The check isn't atomic with BeginShutdown, so a call already past it can still
// dispatch after shutdown starts. Accepted as low-risk: on paths the app controls,
// Fyne is still fully alive at that point; on paths it isn't, the flag is set before
// any service Stop(), i.e. before the cancellation burst that caused the panic this
// guard exists for.
type guardedSignal[T any] struct {
	signals.Signal[T]
	shuttingDown *atomic.Bool
}

func newGuardedSignal[T any](shuttingDown *atomic.Bool) signals.Signal[T] {
	return guardedSignal[T]{signals.New[T](), shuttingDown}
}

func (g guardedSignal[T]) Emit(ctx context.Context, arg T) {
	if g.shuttingDown.Load() {
		return
	}
	g.Signal.Emit(ctx, arg)
}

// UniqueKey returns a unique key for registering listeners.
func (s *Signals) UniqueKey() string {
	return fmt.Sprintf("key-%d", s.keyID.Add(1))
}

func (s *Signals) PseudoUniqueID() string {
	currentTime := time.Now().UnixNano()
	randomNumber := rand.Uint64()
	return fmt.Sprintf("%d-%d", currentTime, randomNumber)
}
