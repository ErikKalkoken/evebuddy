package xgoesi

import (
	"fmt"
	"io"
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
		return resp, nil
	}

	ctx := req.Context()
	tokenSource, ok := ctx.Value(goesi.ContextOAuth2).(oauth2.TokenSource)
	if !ok {
		return resp, nil
	}

	// Always close initial response body before retrying or failing
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	token, err := tokenSource.Token() // refreshes the token when needed
	if err != nil {
		return nil, fmt.Errorf("token refresher transport: %w", err)
	}
	reqClone := req.Clone(ctx)
	token.SetAuthHeader(reqClone)

	return transport.RoundTrip(reqClone)
}
