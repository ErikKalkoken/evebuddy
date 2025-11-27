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

// RateLimitedNonAuth is a wrapper that adds rate limit support to an non-authenticated ESI call.
func RateLimitedNonAuth[T any](operationID string, fetch func() (T, *http.Response, error)) (T, *http.Response, error) {
	return RateLimited(operationID, 0, fetch)
}

// RateLimited is a wrapper that adds rate limit support to an authenticated ESI call.
// It will do nothing if no rate limit is active for an operation.
// It will return an error when an unknown operationID is provided.
func RateLimited[T any](operationID string, characterID int32, fetch func() (T, *http.Response, error)) (T, *http.Response, error) {
	var z T
	group, found := operationID2RateGroupName[operationID]
	if !found {
		return z, nil, fmt.Errorf("operationID not found for rate limits: %s", operationID)
	}
	if group == "" {
		return fetch()
	}
	rl, found := rateLimitGroups[group]
	if !found {
		return z, nil, fmt.Errorf("unknown rate limit group: %s", group)
	}
	delay := rl.windowSize / (time.Duration(rl.maxTokens) / 2)
	delay = time.Duration(float64(delay) * (1 + contingency))
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
