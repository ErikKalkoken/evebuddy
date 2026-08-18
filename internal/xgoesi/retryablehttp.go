package xgoesi

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// CustomCheckRetry is a custom retry policy for a retryablehttp client
// that adds retry for 420s and removes retries for 520s.
func CustomCheckRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	shouldRetry, checkErr := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	if checkErr != nil {
		return false, checkErr
	}
	if resp != nil && resp.StatusCode == StatusUnknownError {
		return false, nil // Don't retry on 520
	}
	if shouldRetry {
		return true, nil
	}
	if resp != nil && resp.StatusCode == StatusTooManyErrors {
		return true, nil // Retry on 420
	}
	return false, nil
}

// CustomBackoff is a custom backoff policy for a retryablehttp client
// that adds backoff for 420s.
func CustomBackoff(minimum time.Duration, maximum time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == StatusTooManyErrors {
		if sleep, ok := ParseErrorLimitResetHeader(resp); ok {
			return sleep
		}
		return ErrorLimitResetFallback
	}
	return retryablehttp.DefaultBackoff(minimum, maximum, attemptNum, resp)
}
