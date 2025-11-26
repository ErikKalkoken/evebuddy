package xesi

import (
	"fmt"
	"time"
)

const contingency = 0.1

type rateLimitGroup struct {
	name       string
	maxTokens  int
	windowSize time.Duration
}

// RateLimitDelayForOperation returns the delay to ensure an average request rate
// stays within a rate limit for an operation (incl. contingency)
// and reports whether the operationID was found.
// If the operation has no rate limit it returns a 0 duration.
func RateLimitDelayForOperation(operationID string) (time.Duration, error) {
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
