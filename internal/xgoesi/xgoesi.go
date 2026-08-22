// Package xgoesi contains extensions to the antihax/goesi library.
package xgoesi

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/fnt-eve/goesi-openapi/esi"
)

var StatusTooManyErrors = 420 // custom status code for ESI
var StatusUnknownError = 520  // custom status code for ESI

// RetryOn401 retries a function f when it returned a 401 error from ESI.
//
// It retries at most maxRetries times. And adds info the log message on retry.
func RetryOn401[T any](ctx context.Context, maxRetries int, info string, f func() (T, error)) (T, error) {
	var retries int
	baseDelay := 100 * time.Millisecond
	maxDelay := 10 * time.Second
	for {
		var shouldRetry bool
		hasChanged, err := f()
		if err != nil {
			if err2, ok := errors.AsType[*esi.GenericOpenAPIError](err); ok {
				if strings.EqualFold(err2.Error(), "401 unauthorized") {
					shouldRetry = true
				}
			}
		}
		if !shouldRetry || retries >= maxRetries {
			return hasChanged, err
		}
		sleepDuration := backoff(retries, baseDelay, maxDelay)
		retries++

		slog.Warn(
			"Retrying update after 401 unauthorized",
			slog.Any("info", info),
			slog.Any("retries", retries),
			slog.Any("maxRetries", maxRetries),
			slog.Any("sleep", sleepDuration),
		)

		select {
		case <-ctx.Done():
			var z T
			return z, ctx.Err()
		case <-time.After(sleepDuration):
		}
	}
}

// backoff returns a backoff duration with equal jitter.
func backoff(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	d := float64(baseDelay) * math.Pow(2, float64(attempt))
	delay := time.Duration(d)

	if delay > maxDelay || delay <= 0 { // overflow check
		delay = maxDelay
	}

	jittered := float64(delay)/2 + (rand.Float64()*float64(delay))/2
	return time.Duration(jittered)
}
