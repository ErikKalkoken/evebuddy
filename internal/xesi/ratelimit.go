package xesi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/antihax/goesi"
	"golang.org/x/time/rate"
)

// TODO: Do thorough play tests
// TODO: Add protection against repeated 429s
// TODO: Consider moving the request logger into own transport to avoid logging cached requests
// TODO: Add protection against repeated 420s

type contextKey string

var (
	contextCharacterID contextKey = "characterID"
	contextOperationID contextKey = "operationID"
)

func (c contextKey) String() string {
	return "xesi " + string(c)
}

// NewContextWithAuth returns a new context has a characterID and an access token set.
func NewContextWithAuth(ctx context.Context, characterID int32, accessToken string) context.Context {
	ctx = context.WithValue(ctx, contextCharacterID, characterID)
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, accessToken)
	return ctx
}

func ContextHasAccessToken(ctx context.Context) bool {
	return ctx.Value(goesi.ContextAccessToken) != nil
}

func NewContextWithOperationID(ctx context.Context, operationID string) context.Context {
	ctx = context.WithValue(ctx, contextOperationID, operationID)
	return ctx
}

type rateLimitGroup struct {
	name       string
	maxTokens  int
	windowSize time.Duration
}

// RateLimiter represents a transport that ensures that the maximum rate of API requests
// complies with the rate limit of the respective operation.
//
// For this feature to work the client must add the operationID of the request
// and the token's character ID to the context.
// It is further assumed that authenticated requests have an access token in the context.
type RateLimiter struct {
	// The RoundTripper interface actually used to make requests
	// If nil, http.DefaultTransport is used
	Transport http.RoundTripper

	mu             sync.Mutex
	limiterBuckets map[string]*rate.Limiter
}

var _ http.RoundTripper = (*RateLimiter)(nil)

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limiterBuckets: make(map[string]*rate.Limiter),
	}
	return rl
}

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
	if hasRateLimit {
		err := rl.wait(ctx, bucket, rlg)
		if err != nil {
			return nil, err
		}
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// wait will wait until the next request can be made to implement a steady request rate
// in accordance with the effective rate limit for the API operation.
func (rl *RateLimiter) wait(ctx context.Context, bucket string, rlg rateLimitGroup) error {
	rl.mu.Lock()
	lim, ok := rl.limiterBuckets[bucket]
	if !ok {
		// calculate the duration to wait for a steady rate assuming each request consumes 2 tokens
		d := rlg.windowSize / (time.Duration(rlg.maxTokens) / 2)
		// add contingency to cover potentially occurring 50x errors which consume additional tokens
		d = time.Duration(float64(d) * (1.1))
		lim = rate.NewLimiter(rate.Every(d), 1)
		rl.limiterBuckets[bucket] = lim
	}
	rl.mu.Unlock()
	if err := lim.Wait(ctx); err != nil {
		return err
	}
	return nil
}

func determineRateLimit(ctx context.Context) (string, rateLimitGroup, bool, error) {
	isAuth := ContextHasAccessToken(ctx)
	operationID, found := ctx.Value(contextOperationID).(string)
	if !found {
		return "", rateLimitGroup{}, false, nil
	}
	characterID, found := ctx.Value(contextCharacterID).(int32)
	if isAuth && !found {
		return "", rateLimitGroup{}, false, fmt.Errorf("ratelimiter: %s: missing character ID for authed request", operationID)
	}
	group, found := operationID2RateGroupName[operationID]
	if !found {
		return "", rateLimitGroup{}, false, fmt.Errorf("ratelimiter: %s: unknown operation", operationID)
	}
	if group == "" {
		return "", rateLimitGroup{}, false, nil
	}
	rlg, found := rateLimitGroups[group]
	if !found {
		return "", rateLimitGroup{}, false, fmt.Errorf("ratelimiter: %s: unknown rate limit group %s", operationID, group)
	}
	bucket := fmt.Sprintf("%s-%d", group, characterID)
	slog.Info("ratelimiter: identified rate limit", "operationID", operationID, "rateLimitGroup", rlg.name, "bucket", bucket) // FIXME: Downgrade to DEBUG for release
	return bucket, rlg, true, nil
}
