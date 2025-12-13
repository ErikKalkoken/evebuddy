package xgoesi

import (
	"context"
	"net/http"
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

func TestParseHeaderForErrorReset(t *testing.T) {
	tests := []struct {
		name        string
		headerValue string
		expectedDur time.Duration
		expectedOK  bool
	}{
		{
			name:        "Success_ValidInteger",
			headerValue: "120",
			expectedDur: time.Second * 120,
			expectedOK:  true,
		},
		{
			name:        "Success_ZeroValue",
			headerValue: "0",
			expectedDur: time.Duration(0),
			expectedOK:  true,
		},
		{
			name:        "Failure_MissingHeader",
			headerValue: "",
			expectedDur: 0,
			expectedOK:  false,
		},
		{
			name:        "Failure_InvalidFormat",
			headerValue: "invalid-duration",
			expectedDur: 0,
			expectedOK:  false,
		},
		{
			name:        "Failure_NegativeDuration",
			headerValue: "-10",
			expectedDur: 0,
			expectedOK:  false,
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

			actualDur, actualBool := ParseErrorLimitResetHeader(resp)

			assert.Equal(t, tt.expectedDur, actualDur, "Returned duration mismatch")
			assert.Equal(t, tt.expectedOK, actualBool, "Returned success boolean mismatch")
		})
	}
}
