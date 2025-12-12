package xgoesi

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/antihax/goesi"
	"golang.org/x/time/rate"
)

// TODO: Add better protection against repeated 429s

var (
	TimeAfter = time.After
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

// rateLimitGroup represents a rate limit group in ESI.
type rateLimitGroup struct {
	name       string
	maxTokens  int
	windowSize time.Duration
}

// rateLimiter represents a transport that ensures that the maximum rate of API requests
// complies with the rate limit of the respective operation.
//
// For this feature to work a client must add the operationID of the request
// and the token's character ID to the context.
// It is further assumed that authenticated requests have an access token in the context.
//
// A rateLimiter must be initialized with [NewRateLimiter] before use.
type rateLimiter struct {
	// The RoundTripper interface actually used to make requests
	// If nil, http.DefaultTransport is used
	Transport  http.RoundTripper
	MaxRetries int

	muRetry420 sync.RWMutex
	retryAt420 time.Time

	muLimiter      sync.Mutex
	limiterBuckets map[string]*rate.Limiter
}

var _ http.RoundTripper = (*rateLimiter)(nil)

// NewRateLimiter returns an initialized RateLimiter.
func NewRateLimiter() *rateLimiter {
	rl := &rateLimiter{
		limiterBuckets: make(map[string]*rate.Limiter),
		MaxRetries:     3,
	}
	return rl
}

func (rl *rateLimiter) RoundTrip(req *http.Request) (*http.Response, error) {
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
		resp, err := rl.roundTripErrorRateLimit(ctx, transport, req)
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

func (rl *rateLimiter) roundTripErrorRateLimit(ctx context.Context, transport http.RoundTripper, req *http.Request) (*http.Response, error) {
	// block when 420 ban active
	rl.muRetry420.RLock()
	retryAfter := time.Until(rl.retryAt420)
	rl.muRetry420.RUnlock()
	if retryAfter > 0 {
		retryAfter = addJitter(retryAfter, time.Second)
		slog.Warn("ESI Error limit block active. Waiting for retry", "url", req.URL, "retryAfter", retryAfter)
		select {
		case <-TimeAfter(retryAfter):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	// make request and retry when encountering 420
	var err error
	var resp *http.Response
	for i := 0; ; i++ {
		var shouldRetry bool
		resp, err = transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == StatusTooManyErrors {
			retryAfter, ok := parseHeaderForErrorReset(resp)
			if !ok {
				slog.Warn("Failed to parse error limit header. Falling back to default.", "url", req.URL)
			}
			rl.muRetry420.Lock()
			rl.retryAt420 = time.Now().Add(retryAfter)
			rl.muRetry420.Unlock()
			if i < rl.MaxRetries {
				retryAfter = addJitter(retryAfter, time.Second)
				slog.Warn("ESI Error limit exceeded. Waiting for retry", "url", req.URL, "retryAfter", retryAfter, "retryNum", i+1)
				select {
				case <-TimeAfter(retryAfter):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				shouldRetry = true
			}
		}
		if shouldRetry {
			continue
		}
		break
	}
	return resp, nil
}

func parseHeaderForErrorReset(resp *http.Response) (time.Duration, bool) {
	const retryAfterFallback = time.Second * 60
	x := resp.Header.Get("X-ESI-Error-Limit-Reset")
	if x == "" {
		return retryAfterFallback, false
	}
	seconds, err := strconv.ParseFloat(x, 64)
	if err != nil {
		return retryAfterFallback, false
	}
	retryAfter := time.Duration(float64(time.Second) * seconds)
	return retryAfter, true
}

func addJitter(d time.Duration, max time.Duration) time.Duration {
	return d + time.Duration(rand.Float64()*float64(max))
}

// waitRateLimit will wait until the next request can be made to implement a steady request rate
// in accordance with the effective rate limit for the API operation.
func (rl *rateLimiter) waitRateLimit(ctx context.Context, bucket string, rlg rateLimitGroup) error {
	rl.muLimiter.Lock()
	lim, ok := rl.limiterBuckets[bucket]
	if !ok {
		// calculate the duration to wait for a steady rate assuming each request consumes 2 tokens
		d := rlg.windowSize / (time.Duration(rlg.maxTokens) / 2)
		// add contingency to cover potentially occurring 50x errors which consume additional tokens
		d = time.Duration(float64(d) * (1.1))
		lim = rate.NewLimiter(rate.Every(d), 1)
		rl.limiterBuckets[bucket] = lim
	}
	rl.muLimiter.Unlock()
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
