package xgoesi

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/fnt-eve/goesi-openapi"
	"golang.org/x/oauth2"
)

// TokenRefresher is a HTTP transport that automatically refreshes expired tokens.
// It covers GET requests only.
//
// When the application is suspended during an update process
// the current token can expire and cause a 401 from the server.
// The TokenRefresher solves this issue by refreshing the token when needed.
type TokenRefresher struct {
	Transport http.RoundTripper
}

func (t *TokenRefresher) RoundTrip(req *http.Request) (*http.Response, error) {
	myLogger := slog.With(
		slog.String("transport", "TokenRefresher"),
		slog.String("method", req.Method),
		slog.Any("url", req.URL),
	)

	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	if req.Method != http.MethodGet {
		myLogger.Warn("Received 401, but can not retry non-GET request")
		return resp, nil
	}

	ctx := req.Context()
	tokenSource, ok := ctx.Value(goesi.ContextOAuth2).(oauth2.TokenSource)
	if !ok {
		myLogger.Warn("Received 401, but context has no token source")
		return resp, nil
	}

	// Always close initial response body before retrying or failing
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if err := ctx.Err(); err != nil {
		myLogger.Warn("Received 401, but context is already done, not attempting refresh", "error", err)
		return nil, err
	}

	myLogger.Info("Received 401, attempting to refresh token")
	token, err := tokenSource.Token() // refreshes the token when needed
	if err != nil {
		myLogger.Warn("Failed to refresh token", "error", err)
		return nil, fmt.Errorf("token refresher transport: %w", err)
	}
	oldAuthHeader := req.Header.Get("Authorization")

	reqClone := req.Clone(ctx)
	token.SetAuthHeader(reqClone)
	myLogger.Info(
		"Refreshed token, retrying request",
		slog.Bool("accessTokenChanged", reqClone.Header.Get("Authorization") != oldAuthHeader),
		slog.Time("expiry", token.Expiry),
	)

	resp2, err := transport.RoundTrip(reqClone)
	if err != nil {
		myLogger.Warn("Retry after token refresh failed", "error", err)
		return resp2, err
	}
	myLogger.Info("Retry after token refresh completed", slog.Int("statusCode", resp2.StatusCode))
	return resp2, nil
}
