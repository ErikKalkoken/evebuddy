package xgoesi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/antihax/goesi"
	"golang.org/x/time/rate"
)

type contextKey string

var (
	contextCharacterID contextKey = "characterID"
	contextOperationID contextKey = "operationID"
)

func (c contextKey) String() string {
	return "xgoesi-" + string(c)
}

// NewContextWithAuth returns a new context with a characterID and an access token.
func NewContextWithAuth(ctx context.Context, characterID int32, accessToken string) context.Context {
	ctx = context.WithValue(ctx, contextCharacterID, characterID)
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, accessToken)
	return ctx
}

// ContextHasAccessToken reports whether the context contains an access token.
func ContextHasAccessToken(ctx context.Context) bool {
	return ctx.Value(goesi.ContextAccessToken) != nil
}

// NewContextWithOperationID returns a new context with a operationID.
func NewContextWithOperationID(ctx context.Context, operationID string) context.Context {
	ctx = context.WithValue(ctx, contextOperationID, operationID)
	return ctx
}

const (
	ErrorLimitResetFallback     = time.Second * 60
	headerErrorLimitRemain      = "X-ESI-Error-Limit-Remain"
	headerErrorLimitReset       = "X-ESI-Error-Limit-Reset"
	headerRetryAfter            = "Retry-After"
	headerRetryAfter429Fallback = 900
	minErrorsRemainDefault      = 5
)

// rateLimitGroup represents a rate limit group in ESI.
type rateLimitGroup struct {
	name       string
	maxTokens  int
	windowSize time.Duration
}

// RateLimiter represents a transport that adds support for ESI rate limits
// and ESI error rate limits.
//
// For the rate limit group detection to work HTTP clients must add the operation ID
// from ESI's OpenAPI spec through [NewContextWithOperationID] to the context.
// Authenticated endpoints must also add the token's character ID
// and access token through [NewContextWithAuth].
// Requests without an operation ID will be assumed to have error rate limiting.
//
// Rate limiting is implemented by ensuring requests belonging to the same bucket
// are not exceeding the average rate of the respective group
// and by temporarily blocking all subsequent requests belonging to the same bucket
// after a 429 status is received from the server.
//
// Error rate limiting is implemented by temporarily blocking all subsequent requests
// after a threshold is reached or after a 420 status is received from the server.
//
// The zero value is a valid transporter.
// This type is designed to be used concurrently.
type RateLimiter struct {
	// The RoundTripper interface actually used to make requests
	// If nil, http.DefaultTransport is used
	Transport http.RoundTripper

	// Minimum number of remaining errors in the current error limit window
	// before blocking subsequent requests.
	MinErrorsRemain int

	muErrors      sync.RWMutex
	retryAtErrors time.Time

	muBuckets      sync.Mutex
	limiterBuckets map[string]*rate.Limiter
	retryAtBuckets map[string]time.Time
}

var _ http.RoundTripper = (*RateLimiter)(nil)

func (rl *RateLimiter) RoundTrip(req *http.Request) (*http.Response, error) {
	myLogger := slog.With(
		slog.String("transport", "RateLimiter"),
		slog.String("method", req.Method),
		slog.Any("url", req.URL),
	)
	transport := rl.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	ctx := req.Context()

	// block during error limit timeout
	rl.muErrors.RLock()
	retryAfterError := time.Until(rl.retryAtErrors)
	rl.muErrors.RUnlock()
	if retryAfterError > 0 {
		retryAfter := int(retryAfterError.Seconds() + 1)
		m := fmt.Sprintf("error limit timeout: %s", retryAfterError)
		resp, err := createErrorResponse(req, StatusTooManyErrors, m)
		if err != nil {
			return nil, err
		}
		resp.Header.Set(headerErrorLimitReset, strconv.Itoa(retryAfter))
		resp.Header.Set(headerErrorLimitRemain, "0")
		myLogger.Warn("Blocked request due to error limit timeout", "retryAfter", retryAfter)
		return resp, nil
	}

	bucket, rlg, hasRateLimit, err := determineRateLimit(ctx)
	if err != nil {
		return nil, err
	}
	myLogger = myLogger.With(
		slog.Bool("hasRatelimit", hasRateLimit),
		slog.String("rateLimitBucket", bucket),
	)
	myLogger.Debug("Processing request")
	if !hasRateLimit {
		// handle error limited operation
		resp, err := transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		switch resp.StatusCode {
		case StatusTooManyErrors:
			// Block all subsequent requests until reset when 420 received
			timeout, ok := ParseErrorLimitResetHeader(resp)
			if !ok {
				myLogger.Warn("Failed to parse error limit header. Using fallback")
				timeout = ErrorLimitResetFallback
			}
			rl.muErrors.Lock()
			rl.retryAtErrors = time.Now().Add(timeout)
			rl.muErrors.Unlock()
			myLogger.Warn("Activated block for ESI error limit. 420 received", "timeout", timeout)
		default:
			// Block all subsequent requests until reset when error threshold is reached
			minErrorsRemain := rl.MinErrorsRemain
			if minErrorsRemain <= 0 || minErrorsRemain >= 100 {
				minErrorsRemain = minErrorsRemainDefault
			}
			if remain, ok := parseErrorLimitRemainHeader(resp); ok {
				if remain <= minErrorsRemain {
					timeout, ok := ParseErrorLimitResetHeader(resp)
					if !ok {
						myLogger.Warn("Failed to parse error limit header. Using fallback.")
						timeout = ErrorLimitResetFallback
					}
					rl.muErrors.Lock()
					rl.retryAtErrors = time.Now().Add(timeout)
					rl.muErrors.Unlock()
					myLogger.Warn("Activated block for ESI error limit. Threshold reached", "timeout", timeout)
				}
			}
		}
		return resp, nil
	}
	// handle rate limited operation
	rl.muBuckets.Lock()
	if rl.retryAtBuckets == nil {
		rl.retryAtBuckets = make(map[string]time.Time)
	}
	retryAfterBucket := time.Until(rl.retryAtBuckets[bucket])
	rl.muBuckets.Unlock()
	if retryAfterBucket > 0 {
		retryAfter := int(retryAfterBucket.Seconds() + 1)
		m := fmt.Sprintf("Rate limit timeout for bucket %s: %s", bucket, retryAfterBucket)
		resp, err := createErrorResponse(req, http.StatusTooManyRequests, m)
		if err != nil {
			return nil, err
		}
		resp.Header.Set(headerRetryAfter, strconv.Itoa(retryAfter))
		myLogger.Warn("Blocked request due to rate limit timeout", "retryAfter", retryAfter)
		return resp, nil
	}
	if err := rl.waitRateLimit(ctx, bucket, rlg); err != nil {
		return nil, err
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		// Block subsequent requests for this bucket when 429 received until retry after
		timeout, ok := parseRetryAfterHeader(resp)
		if !ok {
			myLogger.Warn("Failed to parse retry after header. Using fallback")
			timeout = headerRetryAfter429Fallback
		}
		rl.muBuckets.Lock()
		rl.retryAtBuckets[bucket] = time.Now().Add(timeout)
		rl.muBuckets.Unlock()
		myLogger.Warn("Activated block for rate limit bucket", "timeout", timeout)
	}
	return resp, nil
}

// ParseErrorLimitResetHeader tries to return the value of a ESI error limit reset header
// and reports whether it was successful.
func ParseErrorLimitResetHeader(resp *http.Response) (time.Duration, bool) {
	header := resp.Header.Get(headerErrorLimitReset)
	if header == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(header, 10, 64)
	if err != nil {
		return 0, false
	}
	if v < 0 { // a negative value doesn't make sense
		return 0, false
	}
	return time.Second * time.Duration(v), true
}

// parseErrorLimitRemainHeader tries to return the value of a ESI error limit remain header
// and reports whether it was successful.
func parseErrorLimitRemainHeader(resp *http.Response) (int, bool) {
	header := resp.Header.Get(headerErrorLimitRemain)
	if header == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(header, 10, 64)
	if err != nil {
		return 0, false
	}
	if v < 0 { // a negative value doesn't make sense
		return 0, false
	}
	return int(v), true
}

// parseRetryAfterHeader tries to return the value of standard Retry-After header
// and reports whether it was successful.
func parseRetryAfterHeader(resp *http.Response) (time.Duration, bool) {
	header := resp.Header.Get(headerRetryAfter)
	if header == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(header, 10, 64)
	if err != nil {
		return 0, false
	}
	if v < 0 { // a negative value doesn't make sense
		return 0, false
	}
	return time.Second * time.Duration(v), true
}

func determineRateLimit(ctx context.Context) (string, rateLimitGroup, bool, error) {
	var zero rateLimitGroup
	isAuth := ContextHasAccessToken(ctx)
	operationID, found := ctx.Value(contextOperationID).(string)
	if !found {
		return "", zero, false, nil
	}
	characterID, found := ctx.Value(contextCharacterID).(int32)
	if isAuth && !found {
		return "", zero, false, fmt.Errorf("ratelimiter: %s: missing character ID for authed request", operationID)
	}
	group, found := operationID2RateGroupName[operationID]
	if !found {
		return "", zero, false, fmt.Errorf("ratelimiter: %s: unknown operation", operationID)
	}
	if group == "" {
		return "", zero, false, nil
	}
	rlg, found := rateLimitGroups[group]
	if !found {
		return "", zero, false, fmt.Errorf("ratelimiter: %s: unknown rate limit group %s", operationID, group)
	}
	bucket := fmt.Sprintf("%s-%d", group, characterID)
	return bucket, rlg, true, nil
}

// waitRateLimit will wait until the next request can be made to implement a steady request rate
// in accordance with the effective rate limit for the API operation.
func (rl *RateLimiter) waitRateLimit(ctx context.Context, bucket string, rlg rateLimitGroup) error {
	rl.muBuckets.Lock()
	if rl.limiterBuckets == nil {
		rl.limiterBuckets = make(map[string]*rate.Limiter)
	}
	lim, ok := rl.limiterBuckets[bucket]
	if !ok {
		// calculate the duration to wait for a steady rate assuming each request consumes 2 tokens
		d := rlg.windowSize / (time.Duration(rlg.maxTokens) / 2)
		// add contingency to cover potentially occurring 50x errors which consume additional tokens
		d = time.Duration(float64(d) * (1.1))
		lim = rate.NewLimiter(rate.Every(d), 1)
		rl.limiterBuckets[bucket] = lim
	}
	rl.muBuckets.Unlock()
	if err := lim.Wait(ctx); err != nil {
		return err
	}
	return nil
}
