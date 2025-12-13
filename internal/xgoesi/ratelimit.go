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

// TODO: Add better protection against repeated 429s

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
	ErrorLimitResetFallback = time.Second * 60
	headerErrorLimitReset   = "X-ESI-Error-Limit-Reset"
)

// rateLimitGroup represents a rate limit group in ESI.
type rateLimitGroup struct {
	name       string
	maxTokens  int
	windowSize time.Duration
}

// RateLimiter represents a transport that adds support for rate limits and error rate limits.
//
// For the rate limit group detection to work HTTP clients must add the operation ID
// from ESI's OpenAPI spec through [NewContextWithOperationID]
// and the token's character ID and access token through [NewContextWithAuth] to the context.
// Requests without an operation ID will be assumed to have error rate limiting.
//
// Rate limiting is implemented by ensuring requests belonging to the same bucket
// do not exceed the average rate.
//
// Error rate limiting is implemented by blocking subsequent request
// during the reset period after a 420 is received.
//
// The zero value is a valid transporter.
// This type is designed to be used concurrently.
type RateLimiter struct {
	// The RoundTripper interface actually used to make requests
	// If nil, http.DefaultTransport is used
	Transport http.RoundTripper

	muErrors      sync.RWMutex
	retryAtErrors time.Time

	muBuckets      sync.Mutex
	limiterBuckets map[string]*rate.Limiter
}

var _ http.RoundTripper = (*RateLimiter)(nil)

func (rl *RateLimiter) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := rl.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	ctx := req.Context()
	bucket, rlg, hasRateLimit, err := determineRateLimit(ctx)
	if err != nil {
		return nil, err
	}
	if !hasRateLimit {
		resp, err := rl.roundTripErrorRateLimit(transport, req)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	if err := rl.waitRateLimit(ctx, bucket, rlg); err != nil {
		return nil, err
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rl *RateLimiter) roundTripErrorRateLimit(transport http.RoundTripper, req *http.Request) (*http.Response, error) {
	// block during 420 timeout
	rl.muErrors.RLock()
	retryAfter := time.Until(rl.retryAtErrors)
	rl.muErrors.RUnlock()
	if retryAfter > 0 {
		retryAfterSeconds := int(retryAfter.Seconds() + 1)
		slog.Warn("ESI Error limit timeout active", "url", req.URL, "retryAfter", retryAfterSeconds)
		resp, err := createErrorResponse(req, StatusTooManyErrors, retryAfterSeconds, "Too many errors ban still active")
		if err != nil {
			return nil, err
		}
		resp.Header.Set(headerErrorLimitReset, strconv.Itoa(retryAfterSeconds))
		return resp, nil
	}
	// forward request
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	// handle TooManyErrors response
	if resp.StatusCode == StatusTooManyErrors {
		retryAfter, ok := ParseErrorLimitResetHeader(resp)
		if !ok {
			slog.Warn("Failed to parse error limit header. Using fallback.", "url", req.URL)
			retryAfter = ErrorLimitResetFallback
		}
		rl.muErrors.Lock()
		rl.retryAtErrors = time.Now().Add(retryAfter)
		rl.muErrors.Unlock()
		slog.Warn("ESI Error limit exceeded", "url", req.URL, "retryAfter", retryAfter)
	}
	return resp, nil
}

// ParseErrorLimitResetHeader tries to return the value of a ESI error limit reset header
// and reports if it was successful.
func ParseErrorLimitResetHeader(resp *http.Response) (time.Duration, bool) {
	header := resp.Header.Get(headerErrorLimitReset)
	if header == "" {
		return 0, false
	}
	seconds, err := strconv.ParseInt(header, 10, 64)
	if err != nil {
		return 0, false
	}
	if seconds < 0 { // a negative sleep doesn't make sense
		return 0, false
	}
	return time.Second * time.Duration(seconds), true
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
	slog.Debug("ratelimiter: applying limit", "operationID", operationID, "rateLimitGroup", rlg.name, "bucket", bucket)
	return bucket, rlg, true, nil
}
