package main

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/juju/mutex/v2"
)

type realtime struct{}

func (r realtime) After(d time.Duration) <-chan time.Time {
	c := make(chan time.Time)
	go func() {
		time.Sleep(d)
		c <- time.Now()
	}()
	return c
}

func (r realtime) Now() time.Time {
	return time.Now()
}

// ensureSingleInstance sets and returns a mutex for this application instance.
// The returned mutex must not be released until the application terminates.
func ensureSingleInstance() (mutex.Releaser, error) {
	slog.Info("Checking for other instances")
	mu, err := mutex.Acquire(mutex.Spec{
		Name:    strings.ReplaceAll(appID, ".", "-"),
		Clock:   realtime{},
		Delay:   mutexDelay,
		Timeout: mutexTimeout,
	})
	if errors.Is(err, mutex.ErrTimeout) {
		return nil, fmt.Errorf("another instance running")
	} else if err != nil {
		return nil, fmt.Errorf("acquire mutex: %w", err)
	}
	slog.Info("No other instances running")
	return mu, nil
}
