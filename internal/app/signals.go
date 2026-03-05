package app

import (
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

type GeneralSectionUpdated struct {
	Changed      set.Set[int64]
	Section      GeneralSection
	NeedsRefresh bool
}

// Signals represents the app's event signals.
// It is safe for concurrent use.
type Signals struct {
	// A character was added.
	CharacterAdded signals.Signal[*Character]

	// A character was removed.
	CharacterRemoved signals.Signal[*EntityShort]

	// A character section has changed after an update from ESI.
	CharacterSectionChanged signals.Signal[CharacterSectionUpdated]

	// A character section has been updated from ESI.
	CharacterSectionUpdated signals.Signal[CharacterSectionUpdated]

	// The current character was exchanged with another character or reset.
	CurrentCharacterExchanged signals.Signal[*Character]

	// The current corporation was exchanged with another corporation or reset.
	CurrentCorporationExchanged signals.Signal[*Corporation]

	// A corporation has been added or removed.
	CorporationsChanged signals.Signal[struct{}]

	// A corporation section has changed after an update from ESI.
	CorporationSectionChanged signals.Signal[CorporationSectionUpdated]

	// A corporation section has been updated from ESI.
	CorporationSectionUpdated signals.Signal[CorporationSectionUpdated]

	// A general section has changed after an update from ESI.
	GeneralSectionChanged signals.Signal[GeneralSectionUpdated]

	// A general section has been updated after an update from ESI.
	GeneralSectionUpdated signals.Signal[GeneralSectionUpdated]

	// Ticker for dynamic UI refresh has expired.
	RefreshTickerExpired signals.Signal[struct{}]

	// A tag as been added, removed or renamed.
	TagsChanged signals.Signal[struct{}]

	// A Character has changed [only: trainingWatcher].
	CharacterChanged signals.Signal[int64]

	// A section update has started.
	UpdateStarted signals.Signal[string]

	// A section update has stopped.
	UpdateStopped signals.Signal[string]

	keyID atomic.Uint64
}

func NewSignals() *Signals {
	s := &Signals{
		CharacterAdded:              signals.New[*Character](),
		CharacterChanged:            signals.New[int64](),
		CharacterRemoved:            signals.New[*EntityShort](),
		CharacterSectionChanged:     signals.New[CharacterSectionUpdated](),
		CharacterSectionUpdated:     signals.New[CharacterSectionUpdated](),
		CorporationsChanged:         signals.New[struct{}](),
		CorporationSectionChanged:   signals.New[CorporationSectionUpdated](),
		CorporationSectionUpdated:   signals.New[CorporationSectionUpdated](),
		CurrentCharacterExchanged:   signals.New[*Character](),
		CurrentCorporationExchanged: signals.New[*Corporation](),
		GeneralSectionChanged:       signals.New[GeneralSectionUpdated](),
		GeneralSectionUpdated:       signals.New[GeneralSectionUpdated](),
		RefreshTickerExpired:        signals.New[struct{}](),
		TagsChanged:                 signals.New[struct{}](),
		UpdateStarted:               signals.New[string](),
		UpdateStopped:               signals.New[string](),
	}
	return s
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
