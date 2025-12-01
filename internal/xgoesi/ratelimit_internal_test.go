package xgoesi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	t.Run("should set character ID and access token", func(t *testing.T) {
		ctx := NewContextWithAuth(context.Background(), 42, "token")
		characterID, ok := ctx.Value(contextCharacterID).(int32)
		assert.True(t, ok)
		assert.Equal(t, int32(42), characterID)
		ok2 := ContextHasAccessToken(ctx)
		assert.True(t, ok2)
	})
	t.Run("should report and return operation ID when set", func(t *testing.T) {
		ctx := NewContextWithOperationID(context.Background(), "op")
		id, ok := ctx.Value(contextOperationID).(string)
		assert.True(t, ok)
		assert.Equal(t, "op", id)
	})
}

func TestAddJitter(t *testing.T) {
	tests := []struct {
		name         string
		baseDuration time.Duration
		maxJitter    time.Duration
	}{
		{
			name:         "PositiveBaseAndMax",
			baseDuration: 10 * time.Second,
			maxJitter:    5 * time.Second,
		},
		{
			name:         "ZeroMaxJitter",
			baseDuration: 1 * time.Minute,
			maxJitter:    0 * time.Second,
		},
		{
			name:         "ZeroBaseDuration",
			baseDuration: 0 * time.Second,
			maxJitter:    3 * time.Hour,
		},
		{
			name:         "LargeValues",
			baseDuration: 24 * time.Hour,
			maxJitter:    1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addJitter(tt.baseDuration, tt.maxJitter)

			expectedMin := tt.baseDuration
			expectedMaxExclusive := tt.baseDuration + tt.maxJitter

			assert.GreaterOrEqual(t, got, expectedMin)
			assert.LessOrEqual(t, got, expectedMaxExclusive)

		})
	}
}

func TestParseHeaderForErrorReset(t *testing.T) {
	const retryAfterFallback = time.Second * 60

	tests := []struct {
		name         string
		headerValue  string
		expectedDur  time.Duration
		expectedBool bool
	}{
		{
			name:         "Success_ValidFloat",
			headerValue:  "30.5",
			expectedDur:  time.Second*30 + time.Millisecond*500,
			expectedBool: true,
		},
		{
			name:         "Success_ValidInteger",
			headerValue:  "120",
			expectedDur:  time.Second * 120,
			expectedBool: true,
		},
		{
			name:         "Success_ZeroValue",
			headerValue:  "0",
			expectedDur:  time.Duration(0),
			expectedBool: true,
		},
		{
			name:         "Failure_MissingHeader",
			headerValue:  "",
			expectedDur:  retryAfterFallback,
			expectedBool: false,
		},
		{
			name:         "Failure_InvalidFormat",
			headerValue:  "invalid-duration",
			expectedDur:  retryAfterFallback,
			expectedBool: false,
		},
		{
			name:         "Success_NegativeDuration",
			headerValue:  "-10.0",
			expectedDur:  time.Second * -10,
			expectedBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: make(http.Header),
			}
			if tt.headerValue != "" {
				resp.Header.Set("X-ESI-Error-Limit-Reset", tt.headerValue)
			}

			actualDur, actualBool := parseHeaderForErrorReset(resp)

			assert.Equal(t, tt.expectedDur, actualDur, "Returned duration mismatch")
			assert.Equal(t, tt.expectedBool, actualBool, "Returned success boolean mismatch")
		})
	}
}

func TestRateLimiter_ErrorLimited2(t *testing.T) {
	t.Run("should wait when 420 is blocked", func(t *testing.T) {
		original := TimeAfter
		defer func() {
			TimeAfter = original
		}()
		var callCount int
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
			callCount++
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		var gotRetryAfter time.Duration
		TimeAfter = func(d time.Duration) <-chan time.Time {
			gotRetryAfter = d
			c := make(chan time.Time)
			close(c)
			return c
		}
		rl := NewRateLimiter()
		rl.retryAt420 = time.Now().Add(5 * time.Second)
		client := &http.Client{
			Transport: rl,
		}
		resp, err := client.Get(ts.URL)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.GreaterOrEqual(t, gotRetryAfter, 5*time.Second)
		assert.LessOrEqual(t, gotRetryAfter, 5*time.Second+1*time.Second)
		assert.Equal(t, 1, callCount)
	})
	t.Run("should not wait when 420 block has passed", func(t *testing.T) {
		original := TimeAfter
		defer func() {
			TimeAfter = original
		}()
		var callCount int
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
			callCount++
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		var gotRetryAfter time.Duration
		TimeAfter = func(d time.Duration) <-chan time.Time {
			gotRetryAfter = d
			c := make(chan time.Time)
			close(c)
			return c
		}
		rl := NewRateLimiter()
		rl.retryAt420 = time.Now().Add(-5 * time.Second)
		client := &http.Client{
			Transport: rl,
		}
		resp, err := client.Get(ts.URL)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.EqualValues(t, 0, gotRetryAfter)
		assert.Equal(t, 1, callCount)
	})
}
