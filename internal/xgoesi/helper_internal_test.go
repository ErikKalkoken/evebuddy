package xgoesi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createErrorResponse(t *testing.T) {
	mockReq, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	tests := []struct {
		name              string
		statusCode        int
		retryAfterSeconds int
		message           string
		req               *http.Request
		expectError       bool
		expectedErrSubstr string
	}{
		{
			name:              "Success_400_BadRequest",
			statusCode:        http.StatusBadRequest,
			retryAfterSeconds: 5,
			message:           "Invalid input parameters",
			req:               mockReq,
			expectError:       false,
		},
		{
			name:              "Success_503_ServiceUnavailable",
			statusCode:        http.StatusServiceUnavailable,
			retryAfterSeconds: 60,
			message:           "Server overloaded, try again later",
			req:               mockReq,
			expectError:       false,
		},
		{
			name:              "Failure_StatusCode_TooLow",
			statusCode:        http.StatusOK,
			retryAfterSeconds: 10,
			message:           "Should not matter",
			req:               mockReq,
			expectError:       true,
			expectedErrSubstr: "statusCode must be >=400",
		},
		{
			name:              "Failure_RetryAfter_TooLow",
			statusCode:        http.StatusTooManyRequests,
			retryAfterSeconds: 0,
			message:           "Should not matter",
			req:               mockReq,
			expectError:       true,
			expectedErrSubstr: "retryAfterSeconds must be >0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := createErrorResponse(tt.req, tt.statusCode, tt.retryAfterSeconds, tt.message)

			if tt.expectError {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedErrSubstr)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				assert.Equal(t, tt.statusCode, resp.StatusCode)
				assert.Equal(t, fmt.Sprintf("%d %s", tt.statusCode, http.StatusText(tt.statusCode)), resp.Status)
				assert.Equal(t, tt.req, resp.Request)
				assert.Equal(t, "HTTP/1.1", resp.Proto)
				assert.Equal(t, 1, resp.ProtoMajor)
				assert.Equal(t, 1, resp.ProtoMinor)
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

				assert.Equal(t, strconv.Itoa(tt.retryAfterSeconds), resp.Header.Get("Retry-After"))
				assert.Equal(t, "localhost", resp.Header.Get("X-Origin-Server"))

				bodyBytes, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				defer resp.Body.Close()

				assert.Equal(t, int64(len(bodyBytes)), resp.ContentLength)

				expectedBody := map[string]string{"error": tt.message}
				expectedBodyData, _ := json.Marshal(expectedBody)

				assert.JSONEq(t, string(expectedBodyData), string(bodyBytes))

				var actualBody map[string]string
				unmarshalErr := json.Unmarshal(bodyBytes, &actualBody)
				require.NoError(t, unmarshalErr)
				assert.Equal(t, tt.message, actualBody["error"])

				assert.NoError(t, resp.Body.Close())
			}
		})
	}
}
