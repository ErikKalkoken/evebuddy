package core

import (
	"os"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui/gamesearch"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/screens"
)

func TestMain(m *testing.M) {
	// Leftover goroutines from one test can render text concurrently with the next,
	// which is not safe in Fyne's shared text shaper.
	runAsync = func(f func()) { f() }
	screens.RunAsync = func(f func()) { f() }
	screens.RunAfter = func(time.Duration, func()) {}
	gamesearch.RunAsync = func(f func()) { f() }
	newUpdateStatusThrottle = func(f func()) throttle { return syncThrottle{f} }
	os.Exit(m.Run())
}

// syncThrottle runs f on every trigger.
type syncThrottle struct{ f func() }

func (s syncThrottle) Trigger() { s.f() }
