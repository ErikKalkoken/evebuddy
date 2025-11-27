package xesi

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimited(t *testing.T) {
	tests := []struct {
		name        string
		operationID string
		wantDelay   time.Duration
		wantErr     bool
	}{
		{
			name:        "Valid operation without rate limit",
			operationID: "GetUniverseRegions",
			wantDelay:   0,
			wantErr:     false,
		},
		{
			name:        "Valid operation with rate limit",
			operationID: "GetCharactersCharacterIdMailLists",
			wantDelay:   3300 * time.Millisecond,
			wantErr:     false,
		},
		{
			name:        "Invalid operation",
			operationID: "op_unknown_request",
			wantDelay:   0,
			wantErr:     true,
		},
		{
			name:        "Empty operation ID",
			operationID: "",
			wantDelay:   0,
			wantErr:     true,
		},
	}
	original := sleep
	defer func() {
		sleep = original
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotDelay time.Duration
			sleep = func(d time.Duration) {
				gotDelay = d
			}
			x, _, gotErr := rateLimited(tt.operationID, 42, func() (string, *http.Response, error) {
				return "done", nil, nil
			})
			if tt.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, "done", x)
				assert.Equal(t, tt.wantDelay, gotDelay)
			}
		})
	}
}
