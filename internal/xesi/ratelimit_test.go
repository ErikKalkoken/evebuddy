package xesi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xesi"
)

func TestRateLimitDelayForOperation(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDelay, gotErr := xesi.RateLimitDelayForOperation(tt.operationID)
			if tt.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.Equal(t, tt.wantDelay, gotDelay)
			}
		})
	}
}
