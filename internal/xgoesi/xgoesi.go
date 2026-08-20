// Package xgoesi contains extensions to the antihax/goesi library.
package xgoesi

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/fnt-eve/goesi-openapi/esi"
)

var StatusTooManyErrors = 420 // custom status code for ESI
var StatusUnknownError = 520  // custom status code for ESI

// RetryOn401 retries a function f when it returned a 401 error from ESI.
//
// It retries at most maxRetries times. And adds info the log message on retry.
func RetryOn401[T any](maxRetries int, info string, f func() (T, error)) (T, error) {
	var retries int
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
		slog.Warn(
			"Retrying update after 401 unauthorized",
			slog.Any("info", info),
			slog.Any("retries", retries),
		)
		retries++
	}
}
