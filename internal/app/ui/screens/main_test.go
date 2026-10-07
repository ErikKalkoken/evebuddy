package screens

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Leftover goroutines from one test can render text concurrently with the next,
	// which is not safe in Fyne's shared text shaper.
	runAsync = func(f func()) { f() }
	runAfter = func(time.Duration, func()) {}
	os.Exit(m.Run())
}
