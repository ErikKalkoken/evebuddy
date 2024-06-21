// Package sso provides the ability to authenticate characters with the Eve Online SSO API for desktop apps.
// It implements OAuth 2.0 with the PKCE protocol.
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
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/pkg/browser"
)

type contextKey int

const (
	keyCodeVerifier           contextKey = iota
	keyError                  contextKey = iota
	keyState                  contextKey = iota
	keyAuthenticatedCharacter contextKey = iota
)

const (
	defaultPort            = 30123
	defaultSsoCallbackPath = "/callback"
	host                   = "localhost"
	oauthURL               = "https://login.eveonline.com/.well-known/oauth-authorization-server"
	ssoHost                = "login.eveonline.com"
	ssoIssuer1             = "login.eveonline.com"
	ssoIssuer2             = "https://login.eveonline.com"
	ssoTokenUrl            = "https://login.eveonline.com/v2/oauth/token"
	cacheTimeoutJWKSet     = 6 * 3600
)

var (
	ErrAborted             = errors.New("auth process canceled prematurely")
	ErrTokenError          = errors.New("token error")
	ErrMissingRefreshToken = errors.New("missing refresh token")
)

// Defines a cache service
type CacheService interface {
	Get(any) (any, bool)
	Set(any, any, time.Duration)
}

// SSOService is a service for authentication Eve Online characters.
type SSOService struct {
	// SSO configuration can be modified before calling any method.
	CallbackPath string
	Port         int

	cache      CacheService
	clientID   string
	httpClient *http.Client
}

// Returns a new SSO service.
func New(clientID string, client *http.Client, cache CacheService) *SSOService {
	s := &SSOService{
		Port:         defaultPort,
		CallbackPath: defaultSsoCallbackPath,
		cache:        cache,
		httpClient:   client,
		clientID:     clientID,
	}
	return s
}

// Authenticate an Eve Online character via OAuth 2.0 PKCE and return the new SSO token.
// Will open a new browser tab on the desktop and run a web server for the OAuth process.
func (s *SSOService) Authenticate(ctx context.Context, scopes []string) (*Token, error) {
	codeVerifier, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}
	serverCtx := context.WithValue(ctx, keyCodeVerifier, codeVerifier)
	state, err := generateRandomString(16)
	if err != nil {
		return nil, err
	}
	serverCtx = context.WithValue(serverCtx, keyState, state)
	serverCtx, cancel := context.WithCancel(serverCtx)

	mux := http.NewServeMux()
	mux.HandleFunc(s.CallbackPath, func(w http.ResponseWriter, req *http.Request) {
		slog.Info("Received SSO callback request")
		v := req.URL.Query()
		newState := v.Get("state")
		if newState != serverCtx.Value(keyState).(string) {
			http.Error(w, "Invalid state", http.StatusForbidden)
			return
		}
		code := v.Get("code")
		codeVerifier := serverCtx.Value(keyCodeVerifier).(string)
		rawToken, err := s.retrieveTokenPayload(code, codeVerifier)
		if err != nil {
			msg := "Failed to retrieve token payload"
			slog.Error(msg, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		claims, err := s.validateToken(ctx, rawToken.AccessToken)
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

	if err := s.startSSO(state, codeVerifier, scopes); err != nil {
		return nil, err
	}
	server := &http.Server{
		Addr:    s.address(),
		Handler: mux,
	}
	go func() {
		slog.Info("Web server started", "address", s.address())
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("Web server terminated prematurely", "error", err)
		}
	}()

	<-serverCtx.Done() // wait for the signal to gracefully shutdown the server

	if err := server.Shutdown(context.TODO()); err != nil {
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

func (s *SSOService) address() string {
	return fmt.Sprintf("%s:%d", host, s.Port)
}

func (s *SSOService) redirectURI() string {
	return fmt.Sprintf("http://%s%s", s.address(), s.CallbackPath)
}

// Open browser and show character selection for SSO.
func (s *SSOService) startSSO(state string, codeVerifier string, scopes []string) error {
	challenge, err := calcCodeChallenge(codeVerifier)
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("redirect_uri", s.redirectURI())
	v.Set("client_id", s.clientID)
	v.Set("scope", strings.Join(scopes, " "))
	v.Set("state", state)
	v.Set("code_challenge", challenge)
	v.Set("code_challenge_method", "S256")

	url := fmt.Sprintf("https://login.eveonline.com/v2/oauth/authorize/?%v", v.Encode())
	return browser.OpenURL(url)
}

// Retrieve SSO token from API in exchange for code
func (s *SSOService) retrieveTokenPayload(code, codeVerifier string) (*tokenPayload, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {s.clientID},
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
	resp, err := s.httpClient.Do(req)
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
func (s *SSOService) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, ErrMissingRefreshToken
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {s.clientID},
	}
	rawToken, err := s.fetchOauthToken(form)
	if err != nil {
		return nil, err
	}
	_, err = s.validateToken(ctx, rawToken.AccessToken)
	if err != nil {
		return nil, err
	}
	token := Token{
		AccessToken:  rawToken.AccessToken,
		RefreshToken: rawToken.RefreshToken,
		ExpiresAt:    rawToken.expiresAt(),
	}
	return &token, nil
}

func (s *SSOService) fetchOauthToken(form url.Values) (*tokenPayload, error) {
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

	resp, err := s.httpClient.Do(req)
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

// validateToken validated a JWT token and returns the claims.
// Returns an error when the token is not valid.
func (s *SSOService) validateToken(ctx context.Context, tokenString string) (jwt.MapClaims, error) {
	// parse token and validate signature
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return s.getKey(ctx, t)
	})
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(jwt.MapClaims)
	if err := validateClaims(s.clientID, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// getKey returns the public key for a JWT token.
func (s *SSOService) getKey(ctx context.Context, token *jwt.Token) (any, error) {
	set, err := s.fetchJWKSet(ctx)
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
func (s *SSOService) fetchJWKSet(ctx context.Context) (jwk.Set, error) {
	key := "jwk-set"
	v, found := s.cache.Get(key)
	if found {
		return v.(jwk.Set), nil
	}
	jwksURL, err := s.determineJwksURL()
	if err != nil {
		return nil, err
	}
	set, err := jwk.Fetch(ctx, jwksURL)
	if err != nil {
		return nil, err
	}
	s.cache.Set(key, set, cacheTimeoutJWKSet)
	return set, nil
}

// Determine URL for JWK sets dynamically from web site and return it
func (s *SSOService) determineJwksURL() (string, error) {
	resp, err := s.httpClient.Get(oauthURL)
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

func validateClaims(ssoClientId string, claims jwt.MapClaims) error {
	// validate issuer claim
	iss, err := claims.GetIssuer()
	if err != nil {
		return err
	}
	if iss != ssoIssuer1 && iss != ssoIssuer2 {
		return fmt.Errorf("invalid issuer claim")
	}
	// validate audience claim
	aud, err := claims.GetAudience()
	if err != nil {
		return err
	}
	if aud[0] != ssoClientId {
		return fmt.Errorf("invalid first audience claim")
	}
	if aud[1] != "EVE Online" {
		return fmt.Errorf("invalid 2nd audience claim")
	}
	return nil
}

func calcCodeChallenge(codeVerifier string) (string, error) {
	h := sha256.New()
	_, err := h.Write([]byte(codeVerifier))
	if err != nil {
		return "", err
	}
	bs := h.Sum(nil)
	challenge := base64.RawURLEncoding.EncodeToString(bs)
	return challenge, nil
}

// Generate a random string of given length
func generateRandomString(length int) (string, error) {
	data := make([]byte, length)
	_, err := rand.Read(data)
	if err != nil {
		return "", err
	}
	s := base64.URLEncoding.EncodeToString(data)
	return s, nil
}
