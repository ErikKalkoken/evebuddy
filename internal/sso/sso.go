// Package sso provides the ability to authenticate characters with the Eve Online SSO API.
// It implements the OAuth 2.0 for desktop app with the PKCE protocol.
package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/pkg/browser"

	memcache "github.com/ErikKalkoken/evebuddy/internal/cache"
)

type key int

const (
	keyCodeVerifier           key = iota
	keyError                  key = iota
	keyState                  key = iota
	keyAuthenticatedCharacter key = iota
)

var (
	ErrAborted             = errors.New("auth process canceled prematurely")
	ErrTokenError          = errors.New("token error")
	ErrMissingRefreshToken = errors.New("missing refresh token")
)

const (
	host               = "localhost"
	port               = ":30123"
	address            = host + port
	oauthURL           = "https://login.eveonline.com/.well-known/oauth-authorization-server"
	ssoClientId        = "11ae857fe4d149b2be60d875649c05f1"
	ssoCallbackPath    = "/callback"
	ssoHost            = "login.eveonline.com"
	ssoIssuer1         = "login.eveonline.com"
	ssoIssuer2         = "https://login.eveonline.com"
	ssoTokenUrl        = "https://login.eveonline.com/v2/oauth/token"
	cacheTimeoutJWKSet = 6 * 3600
)

// token payload as returned from SSO API
type tokenPayload struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int32  `json:"expires_in"`
	TokenType        string `json:"token_type"`
	RefreshToken     string `json:"refresh_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

var cache = memcache.New()

// Authenticate an Eve Online character via SSO and return SSO token.
// The process runs in a newly opened browser tab
func Authenticate(ctx context.Context, client *http.Client, scopes []string) (*Token, error) {
	codeVerifier := generateRandomString(32)
	serverCtx := context.WithValue(ctx, keyCodeVerifier, codeVerifier)

	state := generateRandomString(16)
	serverCtx = context.WithValue(serverCtx, keyState, state)

	serverCtx, cancel := context.WithCancel(serverCtx)

	mux := http.NewServeMux()
	mux.HandleFunc(ssoCallbackPath, func(w http.ResponseWriter, req *http.Request) {
		slog.Info("Received SSO callback request")
		v := req.URL.Query()
		newState := v.Get("state")
		if newState != serverCtx.Value(keyState).(string) {
			http.Error(w, "Invalid state", http.StatusForbidden)
			return
		}
		code := v.Get("code")
		codeVerifier := serverCtx.Value(keyCodeVerifier).(string)
		rawToken, err := retrieveTokenPayload(client, code, codeVerifier)
		if err != nil {
			msg := "Failed to retrieve token payload"
			slog.Error(msg, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		claims, err := validateToken(rawToken.AccessToken)
		if err != nil {
			msg := "Failed to validate token"
			slog.Error(msg, "token", rawToken.AccessToken, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		token, err := newToken(rawToken, claims)
		if err != nil {
			msg := "Failed to construct token"
			slog.Error(msg, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		serverCtx = context.WithValue(serverCtx, keyAuthenticatedCharacter, token)
		fmt.Fprintf(
			w,
			"<p>SSO authentication successful for <b>%s</b>.</p><p>You can close this tab now and return to the app.</p>",
			token.CharacterName,
		)
		cancel() // shutdown http server
	})

	if err := startSSO(state, codeVerifier, scopes); err != nil {
		return nil, err
	}
	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}
	go func() {
		slog.Info("Web server started", "address", address)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("Web server terminated prematurely", "error", err)
		}
	}()

	<-serverCtx.Done() // wait for the signal to gracefully shutdown the server

	err := server.Shutdown(context.Background())
	if err != nil {
		return nil, err
	}
	slog.Info("Web server stopped")

	errValue := serverCtx.Value(keyError)
	if errValue != nil {
		return nil, errValue.(error)
	}

	token, ok := serverCtx.Value(keyAuthenticatedCharacter).(*Token)
	if !ok {
		return nil, ErrAborted
	}
	return token, nil
}

// Generate a random string of given length
func generateRandomString(length int) string {
	data := make([]byte, length)
	rand.Read(data)
	v := base64.URLEncoding.EncodeToString(data)
	return v
}

// Open browser and show character selection for SSO
func startSSO(state string, codeVerifier string, scopes []string) error {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("redirect_uri", "http://"+address+ssoCallbackPath)
	v.Set("client_id", ssoClientId)
	v.Set("scope", strings.Join(scopes, " "))
	v.Set("state", state)
	v.Set("code_challenge", calcCodeChallenge(codeVerifier))
	v.Set("code_challenge_method", "S256")

	url := fmt.Sprintf("https://login.eveonline.com/v2/oauth/authorize/?%v", v.Encode())
	err := browser.OpenURL(url)
	return err
}

func calcCodeChallenge(codeVerifier string) string {
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	bs := h.Sum(nil)
	challenge := base64.RawURLEncoding.EncodeToString(bs)
	return challenge
}

// Retrieve SSO token from API in exchange for code
func retrieveTokenPayload(client *http.Client, code, codeVerifier string) (*tokenPayload, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {ssoClientId},
		"code_verifier": {codeVerifier},
	}
	req, err := http.NewRequest(
		"POST",
		"https://login.eveonline.com/v2/oauth/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", ssoHost)

	slog.Info("Sending auth request to SSO API")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, err
	}

	slog.Debug("Response from API", "body", string(body))

	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("details %v, %v: %w", token.Error, token.ErrorDescription, ErrTokenError)
	}
	return &token, nil
}

// Update given token with new instance from SSO API
func RefreshToken(client *http.Client, refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, ErrMissingRefreshToken
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {ssoClientId},
	}
	rawToken, err := fetchOauthToken(client, form)
	if err != nil {
		return nil, err
	}
	token := Token{
		AccessToken:  rawToken.AccessToken,
		RefreshToken: rawToken.RefreshToken,
		ExpiresAt:    calcExpiresAt(rawToken),
	}
	return &token, nil
}

func fetchOauthToken(client *http.Client, form url.Values) (*tokenPayload, error) {
	req, err := http.NewRequest(
		"POST", ssoTokenUrl, strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", "login.eveonline.com")

	slog.Info("Requesting token from SSO API", "grant_type", form.Get("grant_type"), "url", ssoTokenUrl)
	slog.Debug("Request", "form", form)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, err
	}

	slog.Debug("Response from SSO API", "body", string(body))

	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("details %v, %v: %w", token.Error, token.ErrorDescription, ErrTokenError)
	}
	return &token, nil
}

// Validate JWT token and return claims
func validateToken(tokenString string) (jwt.MapClaims, error) {
	// parse token and validate signature
	token, err := jwt.Parse(tokenString, getKey)
	if err != nil {
		return nil, err
	}

	// validate issuer claim
	claims := token.Claims.(jwt.MapClaims)
	iss := claims["iss"].(string)
	if iss != ssoIssuer1 && iss != ssoIssuer2 {
		return nil, fmt.Errorf("invalid issuer claim")
	}

	// validate audience claim
	aud := claims["aud"].([]any)
	if aud[0].(string) != ssoClientId {
		return nil, fmt.Errorf("invalid first audience claim")
	}
	if aud[1].(string) != "EVE Online" {
		return nil, fmt.Errorf("invalid 2nd audience claim")
	}

	return claims, nil
}

// getKey returns the public key for a JWT token.
func getKey(token *jwt.Token) (any, error) {
	set, err := fetchJWKSet()
	if err != nil {
		return nil, err
	}
	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have string kid")
	}

	key, ok := set.LookupKeyID(keyID)
	if !ok {
		return nil, fmt.Errorf("unable to find key %q", keyID)
	}

	var rawKey any
	if err := key.Raw(&rawKey); err != nil {
		return nil, fmt.Errorf("failed to create public key: %s", err)
	}
	return rawKey, nil
}

// fetchJWKSet returns the current JWK set from the web. It is cached.
func fetchJWKSet() (jwk.Set, error) {
	key := "jwk-set"
	v, found := cache.Get(key)
	if found {
		return v.(jwk.Set), nil
	}
	jwksURL, err := determineJwksURL()
	if err != nil {
		return nil, err
	}
	set, err := jwk.Fetch(context.Background(), jwksURL)
	if err != nil {
		return nil, err
	}
	cache.Set(key, set, cacheTimeoutJWKSet)
	return set, nil
}

// Determine URL for JWK sets dynamically from web site and return it
func determineJwksURL() (string, error) {
	resp, err := http.Get(oauthURL)
	if err != nil {
		return "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	jwksURL := data["jwks_uri"].(string)
	return jwksURL, nil
}
