// Package sso provides the ability to authenticate characters with the Eve Online SSO API for desktop apps.
// It implements OAuth 2.0 with the PKCE protocol.
package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type contextKey int

const (
	keyCodeVerifier contextKey = iota
	keyError
	keyState
	keyAuthenticatedCharacter
)

const (
	portDefault         = 30123
	callbackPathDefault = "/callback"
	host                = "localhost"
	protocol            = "http"
	ssoHost             = "login.eveonline.com"
	tokenURLDefault     = "https://login.eveonline.com/v2/oauth/token"
	authorizeURLDefault = "https://login.eveonline.com/v2/oauth/authorize"
	pingTimeout         = 5 * time.Second
)

//go:embed tmpl/*
var templFS embed.FS

var (
	ErrAborted             = errors.New("auth process canceled prematurely")
	ErrTokenError          = errors.New("token error")
	ErrMissingRefreshToken = errors.New("missing refresh token")
)

// SSOService is a service for authentication Eve Online characters.
type SSOService struct {
	// Function to open the default browser. This must to be configured.
	OpenURL func(*url.URL) error
	// When enabled will keep the SSO server running and not start the authentication.
	// This feature is for testing purposes only.
	DemoMode bool

	authorizeURL string
	callbackPath string
	clientID     string
	httpClient   *http.Client
	port         int
	tokenURL     string
}

// New returns a new SSO service.
//
// Important: The OpenURL function must be configured.
func New(clientID string, client *http.Client) *SSOService {
	return new(clientID, client, callbackPathDefault, portDefault, authorizeURLDefault, tokenURLDefault)
}

func new(
	clientID string,
	client *http.Client,
	callbackPath string,
	port int,
	authorizeURL string,
	tokenURL string,
) *SSOService {
	s := &SSOService{
		authorizeURL: authorizeURL,
		callbackPath: callbackPath,
		clientID:     clientID,
		httpClient:   client,
		port:         port,
		tokenURL:     tokenURL,
		OpenURL: func(u *url.URL) error {
			return errors.New("not configured: OpenURL")
		},
	}
	return s
}

// Authenticate authenticates an Eve Online character via OAuth 2.0 PKCE and returns the new SSO token.
//
// Will open a new browser tab on the desktop and run a web server for the OAuth process.
func (s *SSOService) Authenticate(ctx context.Context, scopes []string) (*Token, error) {
	codeVerifier, err := generateRandomStringBase64(32)
	if err != nil {
		return nil, err
	}
	serverCtx := context.WithValue(ctx, keyCodeVerifier, codeVerifier)
	state, err := generateRandomStringBase64(16)
	if err != nil {
		return nil, err
	}
	serverCtx = context.WithValue(serverCtx, keyState, state)
	serverCtx, cancel := context.WithCancel(serverCtx)
	defer cancel()

	processError := func(w http.ResponseWriter, status int, err error) {
		slog.Warn("SSO autentication failed", "error", err)
		http.Error(w, fmt.Sprintf("SSO autentication failed: %s", err), status)
		serverCtx = context.WithValue(serverCtx, keyError, fmt.Errorf("SSO server: %w", err))
		cancel() // shutdown http server
	}

	router := http.NewServeMux()
	// Route for reponding to ping requests
	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "pong\n")
	})
	// Route for stopping the server
	router.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		cancel()
	})
	// Route for responding to SSO callback from CCP server
	router.HandleFunc(s.callbackPath, func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query()
		stateGot := v.Get("state")
		stateWant := serverCtx.Value(keyState).(string)
		if stateGot != stateWant {
			processError(w, http.StatusUnauthorized, fmt.Errorf("invalid state. Want: %s - Got: %s", stateWant, stateGot))
		}
		code := v.Get("code")
		codeVerifier := serverCtx.Value(keyCodeVerifier).(string)
		rawToken, err := s.fetchNewToken(code, codeVerifier)
		if err != nil {
			processError(w, http.StatusUnauthorized, fmt.Errorf("fetch new token: %w", err))
		}
		jwtToken, err := validateJWT(ctx, s.httpClient, rawToken.AccessToken)
		if err != nil {
			processError(w, http.StatusUnauthorized, fmt.Errorf("token validation: %w", err))
		}
		characterID, err := extractCharacterID(jwtToken)
		if err != nil {
			processError(w, http.StatusInternalServerError, fmt.Errorf("extract character ID: %w", err))
		}
		characterName := extractCharacterName(jwtToken)
		scopes := extractScopes(jwtToken)
		token := newToken(rawToken, characterID, characterName, scopes)
		serverCtx = context.WithValue(serverCtx, keyAuthenticatedCharacter, token)
		slog.Info("SSO authentication successful", "characterID", token.CharacterID, "characterName", token.CharacterName)
		http.Redirect(w, r, "/authenticated", http.StatusSeeOther)
	})
	router.HandleFunc("/authenticated", func(w http.ResponseWriter, r *http.Request) {
		var name string
		token, ok := serverCtx.Value(keyAuthenticatedCharacter).(*Token)
		if ok {
			name = token.CharacterName
		} else {
			name = "?"
		}
		t, err := template.ParseFS(templFS, "tmpl/authenticated.html")
		if err != nil {
			processError(w, http.StatusInternalServerError, err)
			return
		}
		err = t.Execute(w, map[string]string{"Name": name})
		if err != nil {
			processError(w, http.StatusInternalServerError, err)
			return
		}
		if s.DemoMode {
			return
		}
		cancel() // shutdown http server
	})
	// Route for returning 404 on all other paths
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	// we want to be sure the server is running before starting the browser
	// and we want to be able to exit early in case the port is blocked
	server := &http.Server{
		Addr:    s.address(),
		Handler: WithLogger(router),
	}
	l, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, fmt.Errorf("SSO server: listen on address: %w", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			slog.Error("SSO server: server close", "error", err)
		}
	}()

	slog.Info("SSO server started", "address", server.Addr)

	go func() {
		if err := server.Serve(l); err != http.ErrServerClosed {
			slog.Error("SSO server: server terminated prematurely", "error", err)
		}
		cancel()
		slog.Info("SSO server stopped")
	}()

	ctxPing, cncl := context.WithTimeout(ctx, pingTimeout)
	defer cncl()

	u := fmt.Sprintf("%s://%s/ping", protocol, server.Addr)
	req, err := http.NewRequestWithContext(ctxPing, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("SSO server: prepare ping: %w", err)
	}
	_, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SSO server: ping: %w", err)
	}

	if !s.DemoMode {
		if err := s.startSSO(state, codeVerifier, scopes); err != nil {
			return nil, fmt.Errorf("SSO server: start SSO: %w", err)
		}
	}
	<-serverCtx.Done()

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Warn("SSO server: server shutdown", "error", err)
	}

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
	return fmt.Sprintf("%s:%d", host, s.port)
}

func (s *SSOService) redirectURI() string {
	return fmt.Sprintf("%s://%s%s", protocol, s.address(), s.callbackPath)
}

// Open browser and show character selection for SSO.
func (s *SSOService) startSSO(state string, codeVerifier string, scopes []string) error {
	challenge, err := calcCodeChallenge(codeVerifier)
	if err != nil {
		return err
	}
	rawURL := s.makeStartURL(challenge, state, scopes)
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	return s.OpenURL(u)
}

func (s *SSOService) makeStartURL(challenge, state string, scopes []string) string {
	v := url.Values{}
	v.Set("client_id", s.clientID)
	v.Set("code_challenge_method", "S256")
	v.Set("code_challenge", challenge)
	v.Set("redirect_uri", s.redirectURI())
	v.Set("response_type", "code")
	v.Set("scope", strings.Join(scopes, " "))
	v.Set("state", state)
	return s.authorizeURL + "/?" + v.Encode()
}

// fetchNewToken returns a new token from SSO API.
func (s *SSOService) fetchNewToken(code, codeVerifier string) (*tokenPayload, error) {
	form := url.Values{
		"client_id":     {s.clientID},
		"code_verifier": {codeVerifier},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequest("POST", s.tokenURL, strings.NewReader(form.Encode()))
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
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("SSO new token: token payload has error: %s, %s: %w", token.Error, token.ErrorDescription, ErrTokenError)
	}
	return &token, nil
}

// Update given token with new instance from SSO API
func (s *SSOService) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, ErrMissingRefreshToken
	}
	rawToken, err := s.fetchRefreshedToken(refreshToken)
	if err != nil {
		return nil, err
	}
	_, err = validateJWT(ctx, s.httpClient, rawToken.AccessToken)
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

func (s *SSOService) fetchRefreshedToken(refreshToken string) (*tokenPayload, error) {
	form := url.Values{
		"client_id":     {s.clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	req, err := http.NewRequest("POST", s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", ssoHost)
	slog.Debug("Requesting token from SSO API", "grant_type", form.Get("grant_type"), "url", s.tokenURL)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("SSO refresh token: token payload has error: %s, %s: %w", token.Error, token.ErrorDescription, ErrTokenError)
	}
	return &token, nil
}

func calcCodeChallenge(codeVerifier string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(codeVerifier)); err != nil {
		return "", err
	}
	challenge := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return challenge, nil
}

// generateRandomStringBase64 returns a random string of given length with base64 encoding.
func generateRandomStringBase64(length int) (string, error) {
	data := make([]byte, length)
	_, err := rand.Read(data)
	if err != nil {
		return "", err
	}
	s := base64.URLEncoding.EncodeToString(data)
	return s, nil
}
