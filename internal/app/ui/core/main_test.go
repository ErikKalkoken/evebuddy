package core

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Leftover goroutines from one test can render text concurrently with the next,
	// which is not safe in Fyne's shared text shaper.
	runAsync = func(f func()) { f() }
	newUpdateStatusThrottle = func(f func()) throttle { return syncThrottle{f} }
	os.Exit(m.Run())
}

// syncThrottle runs f on every trigger.
type syncThrottle struct{ f func() }

func (s syncThrottle) Trigger() { s.f() }
func (s syncThrottle) Stop()    {}
