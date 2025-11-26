package xesi

import (
	"fmt"
	"net/http"
	"time"
)

const contingency = 0.1

type rateLimitGroup struct {
	name       string
	maxTokens  int
	windowSize time.Duration
}

var sleep = time.Sleep

// RateLimited is a wrapper that adds rate limit support to an ESI call.
func RateLimited[T any](operationID string, fetch func() (T, *http.Response, error)) (T, *http.Response, error) {
	var z T
	delay, err := rateLimitDelayForOperation(operationID)
	if err != nil {
		return z, nil, err
	}
	sleep(delay)
	return fetch()
}

// ActivateRateLimiterMock mocks all time delays so that tests can run faster.
func ActivateRateLimiterMock() {
	sleep = func(d time.Duration) {}
}

// DeactivateRateLimiterMock deactivates the time delay mocks.
func DeactivateRateLimiterMock() {
	sleep = time.Sleep
}

// rateLimitDelayForOperation returns the delay to ensure an average request rate
// stays within a rate limit for an operation (incl. contingency)
// and reports whether the operationID was found.
// If the operation has no rate limit it returns a 0 duration.
func rateLimitDelayForOperation(operationID string) (time.Duration, error) {
	group, found := operationID2RateGroupName[operationID]
	if !found {
		return 0, fmt.Errorf("operationID not found for rate limits: %s", operationID)
	}
	rl, found := rateLimitGroups[group]
	if !found {
		return 0, nil
	}
	d := rl.windowSize / (time.Duration(rl.maxTokens) / 2)
	d = time.Duration(float64(d) * (1 + contingency))
	return d, nil
}
