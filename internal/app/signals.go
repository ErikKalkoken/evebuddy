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
	s.AppInit = guardedSignal[struct{}]{signals.New[struct{}](), g}
	s.CharacterAdded = guardedSignal[*Character]{signals.New[*Character](), g}
	s.CharacterChanged = guardedSignal[int64]{signals.New[int64](), g}
	s.CharacterRemoved = guardedSignal[*EntityShort]{signals.New[*EntityShort](), g}
	s.CharacterSectionChanged = guardedSignal[CharacterSectionUpdated]{signals.New[CharacterSectionUpdated](), g}
	s.CharacterSectionUpdated = guardedSignal[CharacterSectionUpdated]{signals.New[CharacterSectionUpdated](), g}
	s.CorporationsChanged = guardedSignal[struct{}]{signals.New[struct{}](), g}
	s.CorporationSectionChanged = guardedSignal[CorporationSectionUpdated]{signals.New[CorporationSectionUpdated](), g}
	s.CorporationSectionUpdated = guardedSignal[CorporationSectionUpdated]{signals.New[CorporationSectionUpdated](), g}
	s.CurrentCharacterExchanged = guardedSignal[*Character]{signals.New[*Character](), g}
	s.CurrentCorporationExchanged = guardedSignal[*Corporation]{signals.New[*Corporation](), g}
	s.DataUpdated = guardedSignal[string]{signals.New[string](), g}
	s.EveUniverseSectionChanged = guardedSignal[EveUniverseSectionUpdated]{signals.New[EveUniverseSectionUpdated](), g}
	s.EveUniverseSectionUpdated = guardedSignal[EveUniverseSectionUpdated]{signals.New[EveUniverseSectionUpdated](), g}
	s.RefreshTickerExpired = guardedSignal[struct{}]{signals.New[struct{}](), g}
	s.TagsChanged = guardedSignal[struct{}]{signals.New[struct{}](), g}
	s.UpdateStarted = guardedSignal[string]{signals.New[string](), g}
	s.UpdateStopped = guardedSignal[string]{signals.New[string](), g}
	return s
}

// BeginShutdown marks the app as shutting down. From this point on, Emit on every
// signal except AppShutdown becomes a no-op, since listeners may touch Fyne widgets
// via fyne.Do, which no longer serializes onto the main thread once Fyne's own quit
// sequence has started draining its dispatch queue.
func (s *Signals) BeginShutdown() {
	s.shuttingDown.Store(true)
}

// IsShuttingDown reports whether BeginShutdown has been called.
func (s *Signals) IsShuttingDown() bool {
	return s.shuttingDown.Load()
}

// guardedSignal wraps a signals.Signal[T] and suppresses Emit once shuttingDown is set.
type guardedSignal[T any] struct {
	signals.Signal[T]
	shuttingDown *atomic.Bool
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
